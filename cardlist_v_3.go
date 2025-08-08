package main

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/forgoer/openssl"
	"github.com/gin-gonic/gin"
)

// Form binds request parameters for encode/decode
type Form struct {
	CardsId [40]int `form:"cards_id"`
	CardsNb [40]int `form:"cards_nb"`
	DeckId  int      `form:"deck_id"`
	HeroId  [3]int   `form:"hero_id"`
	Rank    int      `form:"rank"`
	SkillId [12]int  `form:"skill_id"`
	UID     int      `form:"uid"`
}

// BytesToInt reads a 4-byte little-endian slice into int
func BytesToInt(b []byte) int {
	buf := bytes.NewBuffer(b)
	var x uint32
	binary.Read(buf, binary.LittleEndian, &x)
	return int(x)
}

// ---------------- DECODE FLOW (getDeckList) ---------------- //

// ReverseStringReplacements applies URL-safe base64 to normal base64
func ReverseStringReplacements(msg string) string {
	// 0: '-' -> '+'
	msg = strings.ReplaceAll(msg, "-", "+")
	// 1: '_' -> '/'
	msg = strings.ReplaceAll(msg, "_", "/")
	// 2: '#' -> '='
	msg = strings.ReplaceAll(msg, "#", "=")
	return msg
}

// ReverseProcess decrypts and decompresses raw data
func ReverseProcess(msg string) ([]byte, error) {
	// base64 decode
	b, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return nil, err
	}
	// DES ECB PKCS7 decrypt
	ori, err := openssl.DesECBDecrypt(b, []byte("[&HdN72]"), openssl.PKCS7_PADDING)
	if err != nil {
		return nil, err
	}
	// split off trailer
	dataPart := ori[:len(ori)-4]
	// raw DEFLATE decompress
	r := flate.NewReader(bytes.NewBuffer(dataPart))
	decomp, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	// reverse interleave
	n := len(decomp) / 4
	out := make([]byte, len(decomp))
	for i := 0; i < n; i++ {
		for j := 0; j < 4; j++ {
			out[i*4+j] = decomp[j*n+i]
		}
	}
	return out, nil
}

// getDeckList handles GET and returns decoded deck list
func getDeckList(c *gin.Context) {
	id := c.Param("id")
	url := fmt.Sprintf("https://example.com/deck/%s", id) // adapt endpoint
	resp, err := http.Get(url)
	if err != nil {
		c.String(http.StatusBadGateway, "fetch error: %v", err)
		return
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	msg := ReverseStringReplacements(string(data))
	raw, err := ReverseProcess(msg)
	if err != nil {
		c.String(http.StatusInternalServerError, "decode error: %v", err)
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", raw)
}

// ---------------- ENCODE FLOW (getCodeList) ---------------- //

// IntToBytes converts a uint32 integer to a 4-byte little-endian slice.
func IntToBytes(x int) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint32(x))
	return buf.Bytes()
}

// scrambleBytes applies the interleave for encode
func scrambleBytes(d []byte) []byte {
	n := len(d) / 4
	out := make([]byte, len(d))
	for i := 0; i < n; i++ {
		for j := 0; j < 4; j++ {
			out[j*n+i] = d[i*4+j]
		}
	}
	return out
}

// EncodeProcess applies scramble, compress, encrypt, base64+URL
func EncodeProcess(raw []byte) (string, error) {
	// interleave bytes
	scr := scrambleBytes(raw)
	// raw DEFLATE compress
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.BestCompression)
	w.Write(scr)
	w.Close()
	comp := buf.Bytes()
	// append trailer
	ori := append(comp, IntToBytes(len(scr))...)
	// DES ECB PKCS7 encrypt
	encBytes, err := openssl.DesECBEncrypt(ori, []byte("[&HdN72]"), openssl.PKCS7_PADDING)
	if err != nil {
		return "", err
	}
	// base64 + URL-safe
	enc := base64.StdEncoding.EncodeToString(encBytes)
	enc = strings.NewReplacer(
		"+", "-",
		"/", "_",
		"=", "#",
	).Replace(enc)
	return enc, nil
}

// buildRawFromForm packs Form values into raw byte slice
func buildRawFromForm(form Form) []byte {
	// header: version=0, UID split, Rank, DeckId
	version := 0
	high := form.UID / 100000
	low := form.UID % 100000
	// assemble raw
	headers := []int{version, high, low, form.Rank, form.DeckId}
	// heroes block
	heroesBlock := make([]byte, 4)
	heroesBlock[0] = byte(len(form.HeroId))
	copy(heroesBlock[1:], byteSlice(form.HeroId[:]))
	// skills block: total count in first byte
	totalSkills := len(form.SkillId)
	skillsBlock := make([]byte, 4)
	skillsBlock[0] = byte(totalSkills)
	// pack ints and blocks
	size := len(headers)*4 + len(heroesBlock) + len(skillsBlock) + totalSkills*4 + len(form.CardsId)*4
	raw := make([]byte, size)
	idx := 0
	for _, v := range headers {
		copy(raw[idx:idx+4], IntToBytes(v)); idx += 4
	}
	// heroes
	copy(raw[idx:idx+4], heroesBlock); idx += 4
	// skills count
	copy(raw[idx:idx+4], skillsBlock); idx += 4
	// skill IDs
	for _, s := range form.SkillId {
		copy(raw[idx:idx+4], IntToBytes(s)); idx += 4
	}
	// cards: id + nb
	for i := range form.CardsId {
		code := form.CardsId[i] + ((form.CardsNb[i]&0x0F)<<24)
		copy(raw[idx:idx+4], IntToBytes(code)); idx += 4
	}
	return raw
}

func byteSlice(ints []int) []byte {
	b := make([]byte, len(ints))
	for i, v := range ints {
		b[i] = byte(v)
	}
	return b
}

// getCodeList handles POST and returns encoded deck string
func getCodeList(c *gin.Context) {
	var form Form
	if err := c.ShouldBind(&form); err != nil {
		c.String(http.StatusBadRequest, "bad request: %v", err)
		return
	}
	// build raw from form and encode
	raw := buildRawFromForm(form)
	code, err := EncodeProcess(raw)
	if err != nil {
		c.String(http.StatusInternalServerError, "encode error: %v", err)
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"code": code})
}

func main() {
	r := gin.Default()
	r.GET("/cardlist/getDeckList/:id", getDeckList)
	r.POST("/cardlist/getCodeList", getCodeList)
	r.Run("localhost:8080")
}
