package controller

import (
	"bytes"
	"encoding/binary"
	"strconv"
)

/**
 * 安卓实况照片（Motion Photo / 微动摄影）解析
 *
 * 声明：手机拍出来的实况图就是一个 .jpg —— JPEG 图像数据（到 FFD9 结束）之后
 * 直接追加一段完整的 MP4，所以「普通图片上传」拿到的其实是「封面 + 视频」。
 *
 * 视频起点有两种定位方式：
 *  1. XMP 里的 MicroVideoOffset（Pixel / 小米 / 多数三星）。偏移口径不统一：
 *     有的从文件末尾往前数，有的从文件开头算，所以两个位置都要试一次；
 *  2. 部分机型（三星 SEFF / 华为等）XMP 字段名不一样，直接按 MP4 的 ftyp
 *     盒子扫描兜底。
 *
 * 拆出来的封面与视频分别存成两条附件，前端按「图片::视频」使用
 * （见 Mellow/src/utils/livePhoto.js）。
 */

const (
	// XMP 一定在 JPEG 头部，没必要扫全图
	motionPhotoHeaderScan = 256 * 1024
	// 小于 1KB 的尾巴基本都是误判（缩略图、填充字节）
	motionPhotoMinVideo = 1024
	// 与附件大小上限同量级，防止把异常数据整段当成视频
	motionPhotoMaxVideo = 512 * 1024 * 1024
)

// 不同厂商 / 不同系统版本的 XMP 字段名
var motionPhotoOffsetMarkers = [][]byte{
	[]byte(`MicroVideoOffset="`),
	[]byte(`MotionPhotoOffset="`),
}

var motionPhotoFtypBox = []byte("ftyp")

/**
 * splitMotionPhoto - 从安卓实况照片里拆出封面与视频
 *
 * 不是实况图时 ok 为 false；是实况图但数据异常时同样返回 false，
 * 调用方按普通图片处理即可（图片本身仍能正常显示）。
 *
 * @param data 完整文件内容
 * @return cover 封面（截到视频起始位置），video 视频（MP4），ok 是否拆成功
 */
func splitMotionPhoto(data []byte) (cover []byte, video []byte, ok bool) {

	if len(data) < motionPhotoMinVideo ||
		data[0] != 0xFF || data[1] != 0xD8 {
		return nil, nil, false
	}

	// 1) XMP 偏移（主流机型）
	if offset, found := motionPhotoOffset(data); found {
		for _, start := range motionPhotoStarts(data, offset) {
			if clip := motionPhotoAt(data, start); len(clip) > 0 {
				return data[:start], clip, true
			}
		}
	}

	// 2) 兜底：直接找 MP4 的 ftyp 盒子
	if start := findMp4Box(data); start > 0 {
		if clip := motionPhotoAt(data, start); len(clip) > 0 {
			return data[:start], clip, true
		}
	}

	return nil, nil, false
}

// motionPhotoOffset - 从 XMP 里取 MicroVideoOffset 的数值
func motionPhotoOffset(data []byte) (int, bool) {

	head := data
	if len(head) > motionPhotoHeaderScan {
		head = head[:motionPhotoHeaderScan]
	}

	for _, marker := range motionPhotoOffsetMarkers {

		index := bytes.Index(head, marker)
		if index < 0 {
			continue
		}

		rest := head[index+len(marker):]
		end := bytes.IndexByte(rest, '"')
		if end <= 0 {
			continue
		}

		value, err := strconv.Atoi(string(rest[:end]))
		if err == nil && value > 0 {
			return value, true
		}
	}

	return 0, false
}

// motionPhotoStarts - 偏移量对应的两个候选起点（从文件末尾 / 从文件开头）
func motionPhotoStarts(data []byte, offset int) []int {

	starts := make([]int, 0, 2)

	if fromEnd := len(data) - offset; fromEnd > 2 && fromEnd < len(data) {
		starts = append(starts, fromEnd)
	}

	if offset > 2 && offset < len(data) && (len(starts) == 0 || starts[0] != offset) {
		starts = append(starts, offset)
	}

	return starts
}

// findMp4Box - 在文件里找第一个合法的 MP4 起点（ftyp 盒子的前 4 字节是盒子长度）
func findMp4Box(data []byte) int {

	searchFrom := 2
	for searchFrom < len(data) {

		index := bytes.Index(data[searchFrom:], motionPhotoFtypBox)
		if index < 0 {
			return -1
		}
		// bytes.Index 给的是 ftyp 的位置，盒子起点在它前面 4 字节
		start := searchFrom + index - 4
		if start > 0 && isMp4BoxAt(data, start) {
			return start
		}

		searchFrom += index + len(motionPhotoFtypBox)
	}

	return -1
}

// isMp4BoxAt - start 处是否是一个合法的 ftyp 盒子
func isMp4BoxAt(data []byte, start int) bool {

	if start < 0 || start+8 > len(data) {
		return false
	}

	if string(data[start+4:start+8]) != string(motionPhotoFtypBox) {
		return false
	}

	// ftyp 盒子至少 16 字节：4 字节长度 + "ftyp" + 4 字节主品牌 + 4 字节版本
	size := int(binary.BigEndian.Uint32(data[start : start+4]))
	return size >= 16 && size <= len(data)-start
}

// motionPhotoAt - 校验并截取 start 之后的视频段
func motionPhotoAt(data []byte, start int) []byte {

	if start <= 0 || start >= len(data) || !isMp4BoxAt(data, start) {
		return nil
	}

	clip := data[start:]
	if len(clip) < motionPhotoMinVideo || len(clip) > motionPhotoMaxVideo {
		return nil
	}

	// 真实 MP4 至少要有 moov / mdat 之一，避免把碰巧出现 "ftyp" 的图像数据当视频
	if !bytes.Contains(clip, []byte("moov")) && !bytes.Contains(clip, []byte("mdat")) {
		return nil
	}

	return clip
}
