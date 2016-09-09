package main

import ( 
	"image" 
	"image/color"
	"image/png"
	
	"os"
	"io"
	"bufio"
	"strconv"
	"log"
	"fmt"
	"strings"

	"encoding/binary"

	"flag"
)

var (
	height = flag.Int("height", 1000, "image height")
	width = flag.Int("width", 0, "force image width [overrides line width]")
	lwidth = flag.Int("lwidth", 1, "line width")
	lspace = flag.Int("lspace", 0, "space between lines")
	hmargin = flag.Int("hmargin", 10, "horizontal margin")
	vmargin = flag.Int("vmargin", 10, "vertical margin")
	outName = flag.String("out", "out.png", "output filename")
	dataType = flag.String("type", "text", "datatype: [i8, u8, i16, u16, i32, u32, f32, f64, text]")
	length = flag.Int("length", 1<<31-1, "max number of lines")
	skip = flag.Int("skip", 0, "number of entry to skip")
	endian = flag.String("endian", "little", "endianess: [little, big]")
	gather = flag.String("gather", "max", "data gathering: [max, avg]")
	
	// TODO: back color, front color
)

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func main() {
	flag.Parse()

	if *endian != "little" && *endian != "big" {
		fmt.Printf("invalid endian option '%s'\n", *dataType)
		fmt.Println("usage: hist [options...] <file>")
		flag.PrintDefaults()
		return
	}

	switch *dataType {
	case "text", "i8", "u8", "i16", "u16", "i32", "u32", "f32", "f64":
		break;

	default:
		fmt.Printf("invalid datatype '%s'\n", *dataType)
		fmt.Println("usage: hist [options...] <file>")
		flag.PrintDefaults()
		return
	}
	
	if flag.NArg() == 0 {
		fmt.Println("usage: hist [options...] <file>")
		return
	}

	fmt.Println("scanning input")
	c, min, max, sum, err := analyze(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	vpl := 1.0
	if *width == 0 {
		*width = int((*lwidth + *lspace) * c + *hmargin * 2)
	} else {
		vpl = float64(c) / float64(*width - *hmargin * 2)
		*lwidth = int(1 / vpl) - *lspace
		if *lwidth < 1 {
			*lwidth = 1	// todo: test overflow in image
		}
	}

	zero := .0
	ratio := float64(*height - 2 * *vmargin) / max
	if min < 0 {
		ratio = float64(*height - 2 * *vmargin) / (max - min)
		zero = -min
	}

	fmt.Printf("c=%d min=%f max=%f avg=%f zero=%.2f, vpl=%.2f [%dx%d]\n",
		c, min, max, sum / float64(c), zero, vpl, *width, *height)

	m := image.NewRGBA(image.Rect(0, 0, *width, *height))

	back := color.RGBA {
		R: 255,
		G: 255,
		B: 255,
		A: 255,
	}

	red := color.RGBA {
		R: 255,
		G: 0,
		B: 0,
		A: 255,
	}
	
	fmt.Println("generating background")
	middle := *hmargin + int(zero * ratio)
	for y := 0; y < *height; y += 1 {
		for x := 0; x < *width; x += 1 {
			if y == middle {
				m.Set(x, y, red)
			} else {
				m.Set(x, y, back)
			}
		}
	}

	fmt.Println("processing image")
	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	
	l := 0
	b := bufio.NewReader(f)
	px := 0
	w := *lwidth + *lspace
	h := 0
	v := .0
	s := .0
	for {
		num, err := dataRead(b)
		if err != nil {
			if err == io.EOF {
				break
			} 			
			log.Fatal(err)
		}

		h += 1
		if h < *skip {
			continue
		}

		if *gather == "avg" {
			s += num
		} else {
			if abs(s) < abs(num) {
				s = num
			}
		}

		v += 1
		if v < vpl {
			continue
		}

		if *gather == "avg" {
			num = s / v
		} else {
			num = s
		}
		
		v = 0
		s = 0
		
		lo := int(zero * ratio)
		hi := int((zero + num) * ratio)
		if num < 0 {
			lo = hi
			hi = int(zero * ratio)
		}

		for y := lo; y < hi; y += 1 {
			for i := 0; i < *lwidth; i += 1 {
				px += 1
				m.Set(*hmargin + l * w + i,
					*height - *vmargin - y, red)
			}
		}

		l += 1
		if l >= *length {
			break
		}
	}

	out, err := os.Create(*outName)
	if err != nil {
		log.Fatal(err)
	}

	defer out.Close()

	err = png.Encode(out, m)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s saved %dpx\n", *outName, px)
}

/*

 
*/

func analyze(name string) (int, float64, float64, float64, error) {
	f, err := os.Open(name)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer f.Close()

	var (
		sum, min, max float64
		count int // = 0
	)
	
	b := bufio.NewReader(f)
	h := 0
	for {
		num, err := dataRead(b)
		if err != nil {
			if err == io.EOF {
				break
			} 			
			log.Fatal(err)
		}

		h += 1
		if h < *skip {
			continue
		}

		sum += num

		if count == 0 {
			min = num
			max = num
		}

		if num < min {
			min = num
		}
		
		if num > max {
			max = num
		}
		
		count += 1
		if count >= *length {
			break
		}
	}
	
	return count, min, max, sum, nil
}

func dataRead(buf *bufio.Reader) (float64, error) {
	end := binary.LittleEndian
	if *endian == "little" {
		end = binary.LittleEndian
	}
	
	var num float64 
	switch *dataType {
	case "text":
		line, err := buf.ReadString(10)
		if err != nil {
			return 0, err
		}
		
		num, err = strconv.ParseFloat(
			strings.TrimSpace(line), 64)
		if err != nil {
			return 0, err
		}

	case "i8":
		var i8 int8
		err := binary.Read(buf, end, &i8)
		if err != nil {
			return 0, err
		}

		num = float64(i8)

	case "u8":
		var u8 uint8
		err := binary.Read(buf, end, &u8)
		if err != nil {
			return 0, err
		}

		num = float64(u8)

	case "i16":
		var i16 int16
		err := binary.Read(buf, end, &i16)
		if err != nil {
			return 0, err
		}

		num = float64(i16)

	case "u16":
		var u16 uint16
		err := binary.Read(buf, end, &u16)
		if err != nil {
			return 0, err
		}

		num = float64(u16)

	case "i32":
		var i32 int32
		err := binary.Read(buf, end, &i32)
		if err != nil {
			return 0, err
		}

		num = float64(i32)

	case "u32":
		var u32 uint32
		err := binary.Read(buf, end, &u32)
		if err != nil {
			return 0, err
		}

		num = float64(u32)

	case "f32":
		var f32 float32
		err := binary.Read(buf, end, &f32)
		if err != nil {
			return 0, err
		}

		num = float64(f32)

	case "f64":
		err := binary.Read(buf, end, &num)
		if err != nil {
			return 0, err
		}
	}

	return num, nil
}
