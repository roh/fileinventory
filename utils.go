package main

import (
	"path/filepath"
	"strings"
)

// Data copied from https://github.com/dyne/file-extension-list
var fileExtToType = map[string]string{"ogm": "video", "doc": "document", "class": "code", "js": "code", "swift": "code", "cc": "code", "tga": "image", "ape": "audio", "woff2": "font", "cab": "archive", "whl": "archive", "mpe": "video", "rmvb": "video", "srt": "video", "pdf": "document", "xz": "archive", "m4a": "audio", "vob": "video", "tif": "image", "gz": "archive", "roq": "video", "m4v": "video", "gif": "image", "rb": "code", "3g2": "video", "m4": "code", "ar": "archive", "vb": "code", "sid": "audio", "ai": "image", "wma": "audio", "bmp": "image", "py": "code", "mp4": "video", "m4p": "video", "jpeg": "image", "otf": "font", "ebook": "document", "rtf": "document", "ttf": "font", "ra": "audio", "flv": "video", "ogv": "video", "mpg": "video", "xls": "document", "jpg": "image", "mkv": "video", "nsv": "video", "mp3": "audio", "kmz": "image", "java": "code", "lua": "code", "m2v": "video", "deb": "archive", "rst": "document", "csv": "document", "pls": "audio", "pak": "archive", "egg": "archive", "tlz": "archive", "c": "code", "cbz": "book", "xcodeproj": "code", "iso": "archive", "xm": "audio", "azw": "book", "webm": "video", "3ds": "image", "azw6": "book", "azw3": "book", "php": "code", "kml": "image", "woff": "font", "zipx": "archive", "3gp": "video", "po": "code", "mpa": "audio", "mng": "video", "s7z": "archive", "ics": "document", "go": "code", "ps": "image", "xml": "code", "cpio": "archive", "epub": "document", "docx": "document", "key": "document", "pages": "document", "numbers": "document", "lha": "archive", "flac": "audio", "wmv": "video", "vcxproj": "code", "mar": "archive", "eot": "font", "less": "code", "asf": "video", "apk": "archive", "css": "code", "mp2": "video", "odt": "document", "patch": "code", "wav": "audio", "rs": "code", "gsm": "audio", "ogg": "video", "m": "code", "dds": "image", "h": "code", "dmg": "archive", "mid": "audio", "psd": "image", "procreate": "image", "dwg": "image", "aac": "audio", "s3m": "audio", "cs": "code", "cpp": "code", "au": "audio", "aiff": "audio", "diff": "code", "avi": "video", "html": "code", "txt": "text", "rpm": "archive", "m3u": "audio", "max": "image", "vcf": "document", "svg": "image", "ppt": "document", "clj": "code", "png": "image", "svi": "video", "tiff": "image", "tgz": "archive", "mxf": "video", "7z": "archive", "drc": "video", "yuv": "video", "mov": "video", "tbz2": "archive", "bz2": "archive", "gpx": "image", "shar": "archive", "xcf": "image", "dxf": "image", "jar": "archive", "qt": "video", "tar": "archive", "xpi": "archive", "zip": "archive", "thm": "image", "cxx": "code", "3dm": "image", "rar": "archive", "md": "document", "scss": "code", "mpv": "video", "webp": "image", "war": "archive", "pl": "code", "xlsx": "document", "mpeg": "video", "aaf": "video", "avchd": "video", "mod": "audio", "rm": "video", "it": "audio", "wasm": "code", "el": "code", "eps": "image"}

// GetNormalizedExtension ...
func GetNormalizedExtension(path string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}

// GetFileType ...
func GetFileType(path string) string {
	return fileExtToType[GetNormalizedExtension(path)]
}

// IsHidden ...
func IsHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

// Max for ints
func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
