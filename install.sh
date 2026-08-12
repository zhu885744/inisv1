#!/bin/bash
set -e

# 配置参数（根据实际情况修改）
REMOTE_ZIP_URL="https://cdn.zhuxu.asia/inis-system/inisv1.0.zip"  # 压缩包下载地址
DEPLOY_DIR="/opt/inis"                       # 部署目录
BINARY_NAME="inis_linux_amd64"               # 可执行文件名
SERVICE_NAME="inis"                          # 系统服务名称
APP_PORT="8642"                              # 应用端口

# 颜色与样式定义
RESET="\033[0m"
BOLD="\033[1m"
BLUE="\033[34m"
GREEN="\033[32m"
YELLOW="\033[33m"
RED="\033[31m"
CYAN="\033[36m"
MAGENTA="\033[35m"

# 动画与分隔线
separator() {
    echo -e "${CYAN}=================================================${RESET}"
}

loading() {
    local msg=$1
    local pid=$2
    local spin='⣾⣽⣻⢿⡿⣟⣯⣷'
    local i=0
    echo -n -e "${BOLD}${YELLOW}${msg} "
    while kill -0 $pid 2>/dev/null; do
        i=$(( (i+1) % 8 ))
        echo -n -e "\b${spin:$i:1}"
        sleep 0.1
    done
    echo -e "\b${GREEN}✓${RESET}"
}

# 输出函数
info() {
    echo -e "${GREEN}${BOLD}[INFO]${RESET} $1"
}

warn() {
    echo -e "${YELLOW}${BOLD}[WARN]${RESET} $1"
}

error() {
    echo -e "${RED}${BOLD}[ERROR]${RESET} $1" && exit 1
}

# 启动动画
show_banner() {
    clear
    separator
    echo -e "${MAGENTA}${BOLD}"
    echo "   ██╗███╗   ██╗██╗███████╗  "
    echo "   ██║████╗  ██║██║██╔════╝"
    echo "   ██║██╔██╗ ██║██║███████╗"
    echo "   ██║██║╚██╗██║██║╚════██║"
    echo "   ██║██║ ╚████║██║███████║"
    echo "   ╚═╝╚═╝  ╚═══╝╚═╝╚══════╝"
    echo -e "${RESET}${CYAN}          部署工具 v1.0.0${RESET}"
    separator
    echo
}

# 检查是否已安装
check_installed() {
    if [ -d "$DEPLOY_DIR" ] && [ -f "$DEPLOY_DIR/$BINARY_NAME" ] && [ -f "/etc/systemd/system/$SERVICE_NAME.service" ]; then
        return 0  # 已安装
    else
        return 1  # 未安装
    fi
}

# 显示管理菜单
show_manage_menu() {
    separator
    echo -e "${BLUE}${BOLD}===== 应用管理菜单 =====${RESET}"
    echo "1. 启动服务:   sudo systemctl start $SERVICE_NAME"
    echo "2. 停止服务:   sudo systemctl stop $SERVICE_NAME"
    echo "3. 重启服务:   sudo systemctl restart $SERVICE_NAME"
    echo "4. 查看状态:   sudo systemctl status $SERVICE_NAME"
    echo "5. 查看日志:   journalctl -u $SERVICE_NAME -f"
    echo "6. 设为开机自启: sudo systemctl enable $SERVICE_NAME"
    echo "7. 取消开机自启: sudo systemctl disable $SERVICE_NAME"
    echo "8. 查看应用端口: netstat -tulpn | grep $APP_PORT"
    separator
    echo -e "${YELLOW}${BOLD}提示：你可以直接复制以上命令在终端执行管理操作${RESET}"
    echo
}

# 选择操作（安装/退出）
select_operation() {
    if check_installed; then
        echo -e "${GREEN}${BOLD}检测到应用已安装！${RESET}"
        read -p "是否显示管理脚本？(y/n，默认y): " choice
        choice=${choice:-y}
        if [ "$choice" = "y" ] || [ "$choice" = "Y" ]; then
            show_manage_menu
            exit 0
        else
            info "操作已取消，退出脚本"
            exit 0
        fi
    else
        read -p "未检测到应用，是否开始安装？(y/n，默认y): " choice
        choice=${choice:-y}
        if [ "$choice" != "y" ] && [ "$choice" != "Y" ]; then
            info "安装已取消，退出脚本"
            exit 0
        fi
    fi
}

# 检查依赖工具
check_dependencies() {
    local msg="检查系统依赖"
    (
        local deps=("curl" "unzip" "systemctl")
        for dep in "${deps[@]}"; do
            if ! command -v $dep &> /dev/null; then
                exit 1
            fi
        done
    ) &
    loading "$msg" $!
    if [ $? -ne 0 ]; then
        error "缺失必要依赖，请先安装 curl、unzip、systemctl"
    fi
}

# 创建部署目录
create_deploy_dir() {
    local msg="准备部署目录: $DEPLOY_DIR"
    (
        if [ -d "$DEPLOY_DIR" ]; then
            sudo rm -rf "$DEPLOY_DIR"/*
        else
            sudo mkdir -p "$DEPLOY_DIR"
        fi
        sudo chmod 755 "$DEPLOY_DIR"
    ) &
    loading "$msg" $!
}

# 下载并解压压缩包
download_and_unzip() {
    local msg="下载并解压程序包"
    (
        local temp_zip=$(mktemp)
        if ! curl -fSL "$REMOTE_ZIP_URL" -o "$temp_zip"; then
            rm -f "$temp_zip"
            exit 1
        fi
        sudo unzip -q -o "$temp_zip" -d "$DEPLOY_DIR"
        rm -f "$temp_zip"
        if [ ! -f "$DEPLOY_DIR/$BINARY_NAME" ]; then
            exit 1
        fi
    ) &
    loading "$msg" $!
    if [ $? -ne 0 ]; then
        error "压缩包处理失败，请检查下载地址或文件完整性"
    fi
}

# 设置文件权限
set_permissions() {
    local msg="配置文件权限"
    (
        sudo chmod 755 -R "$DEPLOY_DIR"
        sudo chmod +x "$DEPLOY_DIR/$BINARY_NAME"
        sudo mkdir -p "$DEPLOY_DIR/logs"
        sudo chmod 775 "$DEPLOY_DIR/logs"
    ) &
    loading "$msg" $!
}

# 配置防火墙
configure_firewall() {
    local msg="配置防火墙规则"
    (
        if command -v firewall-cmd &> /dev/null; then
            sudo firewall-cmd --zone=public --add-port="$APP_PORT/tcp" --permanent
            sudo firewall-cmd --reload
        elif command -v ufw &> /dev/null; then
            sudo ufw allow "$APP_PORT/tcp"
        fi
    ) &
    loading "$msg" $!
}

# 创建系统服务
create_system_service() {
    local msg="创建系统服务"
    (
        sudo tee "/etc/systemd/system/$SERVICE_NAME.service" > /dev/null << EOF
[Unit]
Description=inis service
After=network.target

[Service]
WorkingDirectory=$DEPLOY_DIR
ExecStart=$DEPLOY_DIR/$BINARY_NAME
Restart=always
RestartSec=3
StandardOutput=append:$DEPLOY_DIR/logs/app.log
StandardError=append:$DEPLOY_DIR/logs/error.log

[Install]
WantedBy=multi-user.target
EOF
        sudo systemctl daemon-reload
    ) &
    loading "$msg" $!
}

# 启动服务
start_service() {
    local msg="启动应用服务"
    (
        sudo systemctl stop "$SERVICE_NAME" || true
        sudo systemctl start "$SERVICE_NAME"
        sudo systemctl enable "$SERVICE_NAME"
        sleep 2
        if ! sudo systemctl is-active --quiet "$SERVICE_NAME"; then
            exit 1
        fi
    ) &
    loading "$msg" $!
    if [ $? -ne 0 ]; then
        error "服务启动失败，查看日志：journalctl -u $SERVICE_NAME -f"
    fi
}

# 完成提示
show_completion() {
    separator
    echo -e "${GREEN}${BOLD}部署完成！${RESET}"
    echo -e "  ${BOLD}服务状态${RESET}: 运行中"
    echo -e "  ${BOLD}访问地址${RESET}: ${CYAN}http://localhost:$APP_PORT${RESET}"
    echo -e "  ${BOLD}安装引导${RESET}: ${CYAN}http://localhost:$APP_PORT/#/install${RESET}"
    echo -e "  ${BOLD}服务管理${RESET}: systemctl [start|stop|restart|status] $SERVICE_NAME"
    separator
    # 安装完成后也显示管理菜单
    show_manage_menu
}

# 主流程
main() {
    show_banner
    # 新增：选择操作（检测是否安装 + 安装/管理选择）
    select_operation
    check_dependencies
    create_deploy_dir
    download_and_unzip
    set_permissions
    configure_firewall
    create_system_service
    start_service
    show_completion
}

main