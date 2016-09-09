# go-hist
simple go program to draw histograms

Installation
-----------

	go get github.com/xigh/go-hist

Documentation
-----------

	./hist --help
	Usage of ./hist:
	  -endian="little": endianess: [little, big]
	  -gather="max": data gathering: [max, avg]
	  -height=1000: image height
	  -hmargin=10: horizontal margin
	  -length=2147483647: max number of lines
	  -lspace=0: space between lines
	  -lwidth=1: line width
	  -out="out.png": output filename
	  -skip=0: number of entry to skip
	  -type="text": datatype: [i8, u8, i16, u16, i32, u32, f32, f64, text]
	  -vmargin=10: vertical margin
	  -width=0: force image width [overrides line width]
		
