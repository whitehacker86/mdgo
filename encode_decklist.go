package main

import (
	"bufio"
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/forgoer/openssl"
)

// intToBytes converts an integer to 4 bytes (Little Endian)
func intToBytes(x int) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(x))
	return b
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: encode_decklist <input_deck_file>")
		os.Exit(1)
	}
	input := os.Args[1]
	f, err := os.Open(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open %s: %v\n", input, err)
		os.Exit(1)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var (
		version, uid, ladderinfo, attrid int
		heroCount                        int
		heroIDs                          []int
		skillCounts                      []int
		skillIDs                         []int
		cardIDs, cardNums, cardLooks     []int
	)

	// Parse file line by line
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		switch {
		case strings.HasPrefix(line, "version"):
			parts := strings.Split(line, ":")
			version, _ = strconv.Atoi(strings.TrimSpace(parts[1]))

		case strings.HasPrefix(line, "uid"):
			parts := strings.Split(line, ":")
			uid, _ = strconv.Atoi(strings.TrimSpace(parts[1]))

		case strings.HasPrefix(line, "ladderinfo"):
			parts := strings.Split(line, ":")
			ladderinfo, _ = strconv.Atoi(strings.TrimSpace(parts[1]))

		case strings.HasPrefix(line, "attrid"):
			parts := strings.Split(line, ":")
			attrid, _ = strconv.Atoi(strings.TrimSpace(parts[1]))

		case strings.HasPrefix(line, "heroCount"):
			// Format: heroCount: N heroid : id1 id2 ...
			fields := strings.Fields(line)
			heroCount, _ = strconv.Atoi(fields[1])
			for i := 0; i < heroCount; i++ {
				id, _ := strconv.Atoi(fields[4+i])
				heroIDs = append(heroIDs, id)
			}

		case strings.HasPrefix(line, "nskillCount"):
			// Format: nskillCount: N skillCount: c1 c2 ...
			fields := strings.Fields(line)
			for i := 3; i < len(fields); i++ {
				c, _ := strconv.Atoi(fields[i])
				skillCounts = append(skillCounts, c)
			}

		case strings.HasPrefix(line, "skill"):
			// Format: skill : <skillID>
			parts := strings.Split(line, ":")
			sid, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			skillIDs = append(skillIDs, sid)

		case strings.HasPrefix(line, "cardId"):
			// Format: cardId : <id> num: <n> look: <l>
			fields := strings.Fields(line)
			id, _ := strconv.Atoi(fields[2])
			n, _ := strconv.Atoi(fields[4])
			l, _ := strconv.Atoi(fields[6])
			cardIDs = append(cardIDs, id)
			cardNums = append(cardNums, n)
			cardLooks = append(cardLooks, l)
		}
	}

	// Split UID into two parts
	uid1 := uid / 100000
	uid2 := uid % 100000

	// Count skills
	totalSkills := len(skillIDs)

	// Prepare raw data buffer (ints count)
	extraInts := 8 + totalSkills + len(cardIDs)
	d := make([]byte, extraInts*4)

	// Header
	binary.LittleEndian.PutUint32(d[0:4], uint32(version))
	binary.LittleEndian.PutUint32(d[4:8], uint32(uid1))
	binary.LittleEndian.PutUint32(d[8:12], uint32(uid2))
	binary.LittleEndian.PutUint32(d[12:16], uint32(ladderinfo))
	binary.LittleEndian.PutUint32(d[16:20], uint32(attrid))

	// Heroes at int index 5
	base5 := 5 * 4
	d[base5] = byte(heroCount)
	for i := 0; i < len(heroIDs); i++ {
		d[base5+1+i] = byte(heroIDs[i])
	}

	// Skills count at int index 6
	base6 := 6 * 4
	d[base6] = byte(totalSkills)
	for i := 0; i < len(skillCounts) && i < 3; i++ {
		d[base6+1+i] = byte(skillCounts[i])
	}

	// Skill IDs at ints 8..8+totalSkills-1
	for i, sid := range skillIDs {
		offs := (8 + i) * 4
		binary.LittleEndian.PutUint32(d[offs:offs+4], uint32(sid))
	}

	// Card entries after skills
	for i := 0; i < len(cardIDs); i++ {
		id := cardIDs[i]
		n := cardNums[i]
		l := cardLooks[i]
		low := id % 100000
		high := id / 100000
		cardInt := low + (high << 16) + (n << 24) + (l << 28)
		offs := (8 + totalSkills + i) * 4
		binary.LittleEndian.PutUint32(d[offs:offs+4], uint32(cardInt))
	}

	// Inverse byte reorder
	data := make([]byte, len(d))
	for i := 0; i < extraInts; i++ {
		for j := 0; j < 4; j++ {
			data[j*extraInts+i] = d[i*4+j]
		}
	}

	// Compress
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	w.Write(data)
	w.Close()
	comp := buf.Bytes()

	// Append original length
	comp = append(comp, intToBytes(len(data))...)

	// Encrypt DES-ECB (PKCS7)
	encrypted, err := openssl.DesECBEncrypt(comp, []byte("[&HdN72]"), openssl.PKCS7_PADDING)
	if err != nil {
		panic(err)
	}

	// Base64 URL-safe
	s := base64.StdEncoding.EncodeToString(encrypted)
	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "=", "#")

	fmt.Println(s)
}
