@echo off
chcp 936 > nul 2>&1
setlocal enabledelayedexpansion

:: ======================== 极简配置 ========================
:: 直接定义平台参数（避免for/f解析错误）
set "p1_goos=linux"&set "p1_goarch=amd64"&set "p1_out=inis_linux_amd64"&set "p1_desc=Linux x86_64 (云服务器)"
set "p2_goos=linux"&set "p2_goarch=arm64"&set "p2_out=inis_linux_arm64"&set "p2_desc=Linux ARM64 (鲲鹏/树莓派4)"
set "p3_goos=windows"&set "p3_goarch=amd64"&set "p3_out=inis_windows_amd64.exe"&set "p3_desc=Windows x86_64"
set "p4_goos=windows"&set "p4_goarch=arm64"&set "p4_out=inis_windows_arm64.exe"&set "p4_desc=Windows ARM64"
set "p5_goos=darwin"&set "p5_goarch=amd64"&set "p5_out=inis_darwin_amd64"&set "p5_desc=macOS x86_64 (Intel)"
set "p6_goos=darwin"&set "p6_goarch=arm64"&set "p6_out=inis_darwin_arm64"&set "p6_desc=macOS M1/M2"

:: 平台总数
set "PLATFORM_COUNT=6"

:: 默认输出目录
set "DEFAULT_OUTPUT_DIR=./dist"
:: UPX压缩默认等级
set "DEFAULT_COMPRESS_LEVEL=6"

:: ======================== 界面展示 ========================
cls
echo ======================== inis 编译工具 ========================
echo.
echo 请选择要编译的平台：
echo 1 - %p1_desc%
echo 2 - %p2_desc%
echo 3 - %p3_desc%
echo 4 - %p4_desc%
echo 5 - %p5_desc%
echo 6 - %p6_desc%
echo 7 - 所有平台（批量编译）
echo.

:: ======================== 输入选择（极简校验） ========================
set "sel="
:input_loop
set /p "sel=请输入平台编号（1-7）："
if "!sel!"=="" goto input_loop
if !sel! lss 1 goto input_loop
if !sel! gtr 7 goto input_loop
goto input_ok
:input_ok

echo.
:: ======================== 输出目录 ========================
set "OUTPUT_DIR="
set /p "OUTPUT_DIR=请输入编译文件存储目录（默认：%DEFAULT_OUTPUT_DIR%）："
if "!OUTPUT_DIR!"=="" set "OUTPUT_DIR=!DEFAULT_OUTPUT_DIR!"

:: 创建目录
if not exist "!OUTPUT_DIR!" (
    echo [信息] 创建目录：!OUTPUT_DIR!
    md "!OUTPUT_DIR!"
)

echo.
echo 选择的平台：!sel!
echo 输出目录：!OUTPUT_DIR!
echo.

:: ======================== 检查UPX并选择压缩相关配置 ========================
set "COMPRESS_ENABLE=true"
set "COMPRESS_LEVEL="
set "SKIP_COMPRESS=false"

:: 检查UPX是否安装
echo [检查] UPX压缩工具...
upx --version > nul 2>&1
if errorlevel 1 (
    echo [警告] UPX未安装，将跳过压缩步骤（可从https://upx.github.io/下载）。
    set "COMPRESS_ENABLE=false"
) else (
    echo [成功] UPX已安装，支持压缩功能。
    echo.
    :: 新增：选择是否跳过压缩
    :skip_compress_loop
    set "SKIP_COMPRESS_INPUT="
    set /p "SKIP_COMPRESS_INPUT=是否跳过UPX压缩？(y/n，默认：n)："
    if /i "!SKIP_COMPRESS_INPUT!"=="y" set "SKIP_COMPRESS=true"
    if /i "!SKIP_COMPRESS_INPUT!"=="n" set "SKIP_COMPRESS=false"
    if "!SKIP_COMPRESS_INPUT!"=="" set "SKIP_COMPRESS=false"
    :: 校验输入
    if not "!SKIP_COMPRESS_INPUT!"=="" (
        if /i not "!SKIP_COMPRESS_INPUT!"=="y" if /i not "!SKIP_COMPRESS_INPUT!"=="n" goto skip_compress_loop
    )

    if "!SKIP_COMPRESS!"=="true" (
        echo [信息] 已选择跳过UPX压缩
    ) else (
        :: 选择压缩等级
        :level_loop
        set /p "COMPRESS_LEVEL=请输入UPX压缩等级（1-9，1最快9最好，默认：%DEFAULT_COMPRESS_LEVEL%）："
        if "!COMPRESS_LEVEL!"=="" set "COMPRESS_LEVEL=!DEFAULT_COMPRESS_LEVEL!"
        :: 校验压缩等级
        if !COMPRESS_LEVEL! lss 1 goto level_loop
        if !COMPRESS_LEVEL! gtr 9 goto level_loop
        echo [信息] 已选择压缩等级：!COMPRESS_LEVEL!
    )
)
echo.

:: ======================== 环境检查 ========================
echo [检查] Go环境...
go version > nul 2>&1
if errorlevel 1 (
    echo [错误] 未安装Go环境！
    pause
    exit /b 1
)
echo [成功] Go环境正常
echo.

echo [下载] 依赖...
go mod tidy > nul 2>&1
echo [成功] 依赖下载完成
echo.

:: ======================== 编译逻辑 ========================
if "!sel!"=="7" (
    :: ======================== 批量编译所有平台 ========================
    echo ======================== 开始批量编译所有平台 ========================
    echo.
    
    set "total_success=0"
    set "total_fail=0"
    
    for /l %%i in (1,1,!PLATFORM_COUNT!) do (
        set "goos=!p%%i_goos!"
        set "goarch=!p%%i_goarch!"
        set "out=!p%%i_out!"
        set "desc=!p%%i_desc!"
        set "output_file=!OUTPUT_DIR!\!out!"
        
        echo [%%i/!PLATFORM_COUNT!] 正在编译：!desc!
        set CGO_ENABLED=0
        go build -ldflags "-s -w -buildid= -extldflags '-static -s -w'" -trimpath -o "!output_file!" main.go
        
        if errorlevel 1 (
            echo [失败] 编译失败：!desc!
            set /a total_fail+=1
        ) else (
            echo [成功] 编译完成：!output_file!
            set /a total_success+=1
            
            :: 获取文件大小
            for /f "tokens=3" %%f in ('dir /-c "!output_file!" ^| findstr /i "!out!"') do (
                set "size_before=%%f"
                if !size_before! gtr 1048576 (
                    set /a "size_mb=!size_before! / 1048576"
                    set /a "size_mb_remain=!size_before! %% 1048576"
                    set /a "size_mb_decimal=!size_mb_remain! * 100 / 1048576"
                    if !size_mb_decimal! lss 10 set "size_mb_decimal=0!size_mb_decimal!"
                    echo [信息] 压缩前大小：!size_mb!.!size_mb_decimal! MB
                ) else if !size_before! gtr 1024 (
                    set /a "size_kb=!size_before! / 1024"
                    echo [信息] 压缩前大小：!size_kb! KB
                ) else (
                    echo [信息] 压缩前大小：!size_before! 字节
                )
            )
            
            :: UPX压缩
            if "!COMPRESS_ENABLE!"=="true" (
                if "!SKIP_COMPRESS!"=="false" (
                    if not "!goos!"=="darwin" (
                        echo [压缩] 正在压缩...
                        upx -!COMPRESS_LEVEL! --best --lzma --strip-relocs=0 --compress-exports=0 --compress-icons=0 "!output_file!" > nul 2>&1
                        if errorlevel 1 (
                            echo [警告] 压缩失败！
                        ) else (
                            echo [成功] 压缩完成！
                        )
                    ) else (
                        echo [跳过] macOS平台跳过压缩
                    )
                )
            )
        )
        echo.
    )
    
    :: 总结
    echo ======================== 批量编译完成 ========================
    echo [统计] 成功：!total_success! 个，失败：!total_fail! 个
    echo [目录] 所有文件位于：!OUTPUT_DIR!
    
) else (
    :: ======================== 单平台编译 ========================
    echo [编译] 开始编译...
    set "goos=!p%sel%_goos!"
    set "goarch=!p%sel%_goarch!"
    set "out=!p%sel%_out!"
    set "desc=!p%sel%_desc!"
    set "output_file=!OUTPUT_DIR!\!out!"

    echo 编译目标：!desc!
    set CGO_ENABLED=0
    go build -ldflags "-s -w -buildid= -extldflags '-static -s -w'" -trimpath -o "!output_file!" main.go

    if errorlevel 1 (
        echo [失败] 编译失败！
        pause
        exit /b 1
    ) else (
        echo [成功] 编译完成：!output_file!
    )

    :: 显示编译后文件大小
    for /f "tokens=3" %%f in ('dir /-c "!output_file!" ^| findstr /i "!out!"') do (
        set "size_before=%%f"
        if !size_before! gtr 1048576 (
            set /a "size_mb=!size_before! / 1048576"
            set /a "size_mb_remain=!size_before! %% 1048576"
            set /a "size_mb_decimal=!size_mb_remain! * 100 / 1048576"
            if !size_mb_decimal! lss 10 set "size_mb_decimal=0!size_mb_decimal!"
            echo [信息] 压缩前文件大小：!size_mb!.!size_mb_decimal! MB
        ) else if !size_before! gtr 1024 (
            set /a "size_kb=!size_before! / 1024"
            set /a "size_kb_remain=!size_before! %% 1024"
            set /a "size_kb_decimal=!size_kb_remain! * 100 / 1024"
            if !size_kb_decimal! lss 10 set "size_kb_decimal=0!size_kb_decimal!"
            echo [信息] 压缩前文件大小：!size_kb!.!size_kb_decimal! KB
        ) else (
            echo [信息] 压缩前文件大小：!size_before! 字节
        )
    )

    :: UPX压缩
    if "!COMPRESS_ENABLE!"=="true" (
        if "!SKIP_COMPRESS!"=="false" (
            if not "!goos!"=="darwin" (
                echo.
                echo [开始] UPX压缩（等级：!COMPRESS_LEVEL!）...
                upx -!COMPRESS_LEVEL! --best --lzma --strip-relocs=0 --compress-exports=0 --compress-icons=0 "!output_file!" > nul 2>&1
                if errorlevel 1 (
                    echo [警告] 压缩失败！
                ) else (
                    echo [成功] 压缩完成：!output_file!
                    :: 显示压缩后文件大小
                    for /f "tokens=3" %%f in ('dir /-c "!output_file!" ^| findstr /i "!out!"') do (
                        set "size_after=%%f"
                        :: 计算压缩率
                        set /a "compress_rate=(!size_before! - !size_after!) * 100 / !size_before!"
                        if !size_after! gtr 1048576 (
                            set /a "size_mb=!size_after! / 1048576"
                            set /a "size_mb_remain=!size_after! %% 1048576"
                            set /a "size_mb_decimal=!size_mb_remain! * 100 / 1048576"
                            if !size_mb_decimal! lss 10 set "size_mb_decimal=0!size_mb_decimal!"
                            echo [信息] 压缩后文件大小：!size_mb!.!size_mb_decimal! MB（压缩率：!compress_rate!%%）
                        ) else if !size_after! gtr 1024 (
                            set /a "size_kb=!size_after! / 1024"
                            set /a "size_kb_remain=!size_after! %% 1024"
                            set /a "size_kb_decimal=!size_kb_remain! * 100 / 1024"
                            if !size_kb_decimal! lss 10 set "size_kb_decimal=0!size_kb_decimal!"
                            echo [信息] 压缩后文件大小：!size_kb!.!size_kb_decimal! KB（压缩率：!compress_rate!%%）
                        ) else (
                            echo [信息] 压缩后文件大小：!size_after! 字节（压缩率：!compress_rate!%%）
                        )
                    )
                )
            ) else (
                echo.
                echo [提示] macOS平台跳过UPX压缩（避免程序运行异常）。
            )
        ) else (
            echo.
            echo [信息] 已手动跳过UPX压缩步骤
        )
    )
)

:: ======================== 完成 ========================
echo.
echo ======================== 操作完成 ========================
if "!sel!"=="7" (
    echo [成功] 已编译所有平台，文件位于：!OUTPUT_DIR!
) else (
    echo [成功] 最终文件：!output_file!
)
echo 按任意键退出...
pause > nul
exit /b 0