package main

import (
	"fmt"
	"strings"
	"encoding/binary"
	"io/ioutil"
	"bytes"
  	"encoding/base64"
	"compress/flate"
	"github.com/forgoer/openssl"
)

func BytesToInt(b []byte) int {
    bytesBuffer := bytes.NewBuffer(b)

    var x uint32
    binary.Read(bytesBuffer, binary.LittleEndian, &x)

    return int(x)
}

func main() {
	var msg = ("4ikbUvVq2NvaYrqyy72J1Wmzv4HyEncE2j9waUm3cWPSx2SUGrmkIQEewaIY7vMo15iQqeTSIq44DI3Hmqu-CBGuS06-Bi-jMA-EpOi_D_KX5nipidS4OHFwpLy3QTnZup8Me6n9q89F9Yf2U_L4O8JMUkGfwmI4oEvaVR6eohcriImG3yjxoQ##")
	msg = strings.Replace(msg, "-", "+", -1)
	msg = strings.Replace(msg, "_", "/", -1)
	msg = strings.Replace(msg, "#", "=", -1)
	b, e := base64.StdEncoding.DecodeString(msg)
    if(e!=nil){
    	fmt.Println(e)
    }
	//fmt.Println(b[0], b[1], b[2], b[3])
	ori, e := openssl.DesECBDecrypt(b, []byte("[&HdN72]"), openssl.PKCS7_PADDING)
    if(e!=nil){
    	fmt.Println(e)
    }
	fmt.Println(len(ori))
	//fmt.Println(ori[0], ori[1], ori[2], ori[3])
	flateReader := flate.NewReader(bytes.NewBuffer(ori[0:len(ori) - 4]))
	data, e := ioutil.ReadAll(flateReader)
    if(e!=nil){
    	fmt.Println(e)
    }
	fmt.Println(len(data))
	isec := len(data) / 4
	d := make([]byte, len(data))
	for i:=0;i<isec;i++ {
		for j:=0;j<4;j++ {
			d[i*4 + j] = data[j * isec +i]
		}
	}
	fmt.Println("CardsData")
	unpackFn := func(idx int) int{
		return BytesToInt(d[idx*4:idx*4+4])
	}
	fmt.Printf("version : %d \n", unpackFn(0))
	
	fmt.Printf("uid : %d \n", unpackFn(1)*100000 + unpackFn(2))
	fmt.Printf("ladderinfo : %d \n", unpackFn(3))
	fmt.Printf("attrid : %d \n", unpackFn(4))
	fmt.Printf("heroCount: %d heroid : %d %d %d\n", d[5 * 4], d[5 * 4 + 1], d[5 * 4 + 2], d[5 * 4 + 3])
	fmt.Printf("nskillCount: %d skillCount: %d %d %d\n", d[6 * 4], d[6 * 4 + 1], d[6 * 4 + 2], d[6 * 4 + 3])
	for i := 8; i < 8 + int(d[6 *4 + 1]) + int(d[6 * 4 + 2]) + int(d[6 * 4 + 3]); i++ {
		fmt.Printf("skill : %d\n", unpackFn(i))
	}
	for i := 8 + int(d[6 *4 + 1]) + int(d[6 * 4 + 2]) + int(d[6 * 4 + 3]); i < isec; i++ {
		card := unpackFn(i)
		fmt.Printf("cardId : %d num: %d look: %d\n", (card&0xffff)+ ((card&0xff0000) >>16)*100000, (card&0xf000000) >>24 , (card&0xf0000000) >>28 )
	}
}