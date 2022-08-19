package hachibi

import "image"

const (
	DefaultWidthCompression   = uint(3)
	DefaultHeightCompression  = uint(4)
	DefaultQualityCompression = int(50)
)

func main() {

	//img, err := Compress() // default value
	//_ = img
	//_ = err
	//
	//Compress(WithQualityCompression(100)) // quality 100
	//
	//Compress(WithResizeValue(uint(1), uint(5))) // w = 1, h = 5
	//
	//Compress(
	//	WithResizeValue(uint(1), uint(5)),
	//	WithQualityCompression(30),
	//	)
	//
	//
	//var img *image.Image
	//switch  {
	//case <2MB:
	//	im, err := Compress()
	//	img = im
	//}

	Coba(func(i int, i2 int) int {
		return i * i2
	}, 3, 4)

	Coba(func(i int, i2 int) int {
		return i / i2
	}, 3, 4)
}

type QualityOptions struct {
	quality int
}
type QualityOption func(options *QualityOptions)

func WithQualityValue(v int) QualityOption {
	return func(options *QualityOptions) {
		options.quality = v
	}
}

type CompressOptions struct {
	compressQuality bool
	resizeImage     bool

	quality int
	width   uint
	height  uint
}

type CompressOption func(options *CompressOptions)

func Compress(opts ...CompressOption) (*image.Image, error) {
	option := CompressOptions{
		compressQuality: false,
		resizeImage:     false,
	}
	for _, opt := range opts {
		opt(&option)
	}

	if option.compressQuality {
		// todo compress quality
	}

	if option.resizeImage {
		//todo resize image
	}

	return nil, nil
}

func Coba(a func(int, int) int, b int, c int) int {
	return a(b, c) / 2
}

func WithCompressQuality(opts ...QualityOption) CompressOption {
	return func(options *CompressOptions) {
		q := QualityOptions{quality: DefaultQualityCompression}
		for _, opt := range opts {
			opt(&q)
		}

		options.compressQuality = true
		options.quality = q.quality
	}
}

func WithResizeImage() CompressOption {
	return func(options *CompressOptions) {
		options.resizeImage = true
	}
}

func CompressA(w int, h int, q int) {

}
