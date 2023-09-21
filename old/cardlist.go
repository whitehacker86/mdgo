package main

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/forgoer/openssl"
	"github.com/gin-gonic/gin"
)

type deck struct {
    id     int  `json:"uid"`
    rk     int  `json:"rank"`
	did    int  `json:"did"`
}



func BytesToInt(b []byte) int {
    bytesBuffer := bytes.NewBuffer(b)

    var x uint32
    binary.Read(bytesBuffer, binary.LittleEndian, &x)

    return int(x)
}

func getDeckList(c *gin.Context) {
	var msg = c.Param("id")
	var imsg = "66M_udUNWyIyEPbQMOpcMx7B4La_AhuM9Zk65UiR7kQi1HnLnLyH4a2eCQrN98L-PHvGapezWwB3VDFK-SUK2zMLzQAGHKwHlz1HitFJqek1C_P1IpYWLsJC8mg4tGTio0MazOZzVIPI0jg7hn1D01lVDDgc_mcai4YMeq3AFrmgsh-3Oxe-U2OtoVC6XuWgcehfHXNpr4unS-IfB4WB1xgnC6ZpoWcw" //
	
	fmt.Printf("\n MSG  : \n %v \n" , msg)
	fmt.Printf("\n IMSG : \n %v \n" , imsg)


	if (msg == imsg) {
		fmt.Printf("\n MSG == IMSG")
	}
	if (msg != imsg) {
		fmt.Printf("\n MSG != IMSG")
	}

	//fmt.Printf("\n STEP 1 : %v \n" , msg)
	fmt.Printf("\n MSG STEP 0 : \n %v \n" , msg)
	msg = strings.Replace(msg, "-", "+", -1)
	fmt.Printf("\n MSG STEP 0.1 : \n %v \n" , msg)
	msg = strings.Replace(msg, "_", "/", -1)
	fmt.Printf("\n MSG STEP 0.2 : \n %v \n" , msg)
	msg = strings.Replace(msg, "#", "=", -1)
	
	fmt.Printf("\n MSG STEP 1 : \n %v \n" , msg)
	b, err := base64.StdEncoding.DecodeString(msg)
    if(err!=nil){		
    	fmt.Println(err)
		return
    }
	
	ori, err := openssl.DesECBDecrypt(b, []byte("[&HdN72]"), openssl.PKCS7_PADDING)
    if(err!=nil){
    	fmt.Println(err)
		return
    }	
	
	flateReader := flate.NewReader(bytes.NewBuffer(ori[0:len(ori) - 4]))
	data, e := ioutil.ReadAll(flateReader)
    if(e!=nil){
    	fmt.Println(e)
    }
	
	isec := len(data) / 4
	d := make([]byte, len(data))
	fmt.Println(len(data))
	for i:=0;i<isec;i++ {
		for j:=0;j<4;j++ {
			d[i*4 + j] = data[j * isec +i]
		}
	}
	fmt.Println("Data",d)
	
	unpackFn := func(idx int) int{
		return BytesToInt(d[idx*4:idx*4+4])
	}

	//VERSION
	var version = unpackFn(0)
	fmt.Printf("version : %v\n", version)
	
	//UID 
	var uid = unpackFn(1)*100000 + unpackFn(2)
	fmt.Printf("uid : %v\n", uid)
	
	//RANK
	var rank = unpackFn(3)
	fmt.Printf("ladderinfo : %v\n", rank)
	
	//DECK ID
	var dckid = unpackFn(4)
	fmt.Printf("attrid : %v\n", dckid)
	
	//HEROES
	var hero1 = d[5 * 4 + 1]
	var hero2 = d[5 * 4 + 2]
	var hero3 = d[5 * 4 + 3]
	
	fmt.Printf("heroCount: %d heroid : %d %d %d\n", d[5 * 4], hero1 , hero2 , hero3 )
	
	
	//SKILL
	var nbSkillH1 = d[6 * 4 + 1]
	var nbSkillH2 = d[6 * 4 + 2]
	var nbSkillH3 = d[6 * 4 + 3]
	fmt.Printf("nskillCount: %d skillCount: %d %d %d\n", d[6 * 4], nbSkillH1, nbSkillH2, nbSkillH3)
	
	var idSkill [12]int
	for i := 8; i < 8 + int(d[6 *4 + 1]) + int(d[6 * 4 + 2]) + int(d[6 * 4 + 3]); i++ {
		idSkill[i-8] = unpackFn(i)
		fmt.Printf("skill : %d\n", unpackFn(i))
	}
	
	var idCard [40]int
	var nbCard [40]int
	var lookCard [40]int
	var j = 0
	for i := 8 + int(d[6 *4 + 1]) + int(d[6 * 4 + 2]) + int(d[6 * 4 + 3]); i < isec; i++ {
		card := unpackFn(i)
		
		idCard[j] = (card&0xffff)+ ((card&0xff0000) >>16)*100000
		nbCard[j] = (card&0xf000000) >>24
		lookCard[j] = (card&0xf0000000) >>28
		fmt.Printf("cardId : %d num: %d look: %d cardinfo : %v \n", idCard[j], nbCard[j] , lookCard[j] , ((card&0xf000000)>>24&0xf000000) )
		j++
	}
	
	var dlst deck
	dlst.id = uid
	dlst.rk = rank
	dlst.did = dckid
	c.IndentedJSON(200, 
		gin.H{
			"uid": dlst.id,
			"rank": dlst.rk,
			"deck_id" : dlst.did,
			"hero_id01" : hero1,
			"hero_id02" : hero2,
			"hero_id03" : hero3,
			"skill_id01" : idSkill[0],
			"skill_id02" : idSkill[1],
			"skill_id03" : idSkill[2],
			"skill_id04" : idSkill[3],
			"skill_id05" : idSkill[4],
			"skill_id06" : idSkill[5],
			"skill_id07" : idSkill[6],
			"skill_id08" : idSkill[7],
			"skill_id09" : idSkill[8],
			"skill_id10" : idSkill[9],
			"skill_id11" : idSkill[10],
			"skill_id12" : idSkill[11],
			"card_id01" : idCard[0],
			"card_nb01" : nbCard[0],
			"card_id02" : idCard[1],
			"card_nb02" : nbCard[1],
			"card_id03" : idCard[2],
			"card_nb03" : nbCard[2],
			"card_id04" : idCard[3],
			"card_nb04" : nbCard[3],
			"card_id05" : idCard[4],
			"card_nb05" : nbCard[4],
			"card_id06" : idCard[5],
			"card_nb06" : nbCard[5],
			"card_id07" : idCard[6],
			"card_nb07" : nbCard[6],
			"card_id08" : idCard[7],
			"card_nb08" : nbCard[7],
			"card_id09" : idCard[8],
			"card_nb09" : nbCard[8],
			"card_id10" : idCard[9],
			"card_nb10" : nbCard[9],
			"card_id11" : idCard[10],
			"card_nb11" : nbCard[10],
			"card_id12" : idCard[11],
			"card_nb12" : nbCard[11],
			"card_id13" : idCard[12],
			"card_nb13" : nbCard[12],
			"card_id14" : idCard[13],
			"card_nb14" : nbCard[13],
			"card_id15" : idCard[14],
			"card_nb15" : nbCard[14],
			"card_id16" : idCard[15],
			"card_nb16" : nbCard[15],
			"card_id17" : idCard[16],
			"card_nb17" : nbCard[16],
			"card_id18" : idCard[17],
			"card_nb18" : nbCard[17],
			"card_id19" : idCard[18],
			"card_nb19" : nbCard[18],
			"card_id20" : idCard[19],
			"card_nb20" : nbCard[19],
			"card_id21" : idCard[20],
			"card_nb21" : nbCard[20],
			"card_id22" : idCard[21],
			"card_nb22" : nbCard[21],
			"card_id23" : idCard[22],
			"card_nb23" : nbCard[22],
			"card_id24" : idCard[23],
			"card_nb24" : nbCard[23],
			"card_id25" : idCard[24],
			"card_nb25" : nbCard[24],
			"card_id26" : idCard[25],
			"card_nb26" : nbCard[25],
			"card_id27" : idCard[26],
			"card_nb27" : nbCard[26],
			"card_id28" : idCard[27],
			"card_nb28" : nbCard[27],
			"card_id29" : idCard[28],
			"card_nb29" : nbCard[28],
			"card_id30" : idCard[29],
			"card_nb30" : nbCard[29],
			"card_id31" : idCard[30],
			"card_nb31" : nbCard[30],
			"card_id32" : idCard[31],
			"card_nb32" : nbCard[31],
			"card_id33" : idCard[32],
			"card_nb33" : nbCard[32],
			"card_id34" : idCard[33],
			"card_nb34" : nbCard[33],
			"card_id35" : idCard[34],
			"card_nb35" : nbCard[34],
			"card_id36" : idCard[35],
			"card_nb36" : nbCard[35],
			"card_id37" : idCard[36],
			"card_nb37" : nbCard[36],
			"card_id38" : idCard[37],
			"card_nb38" : nbCard[37],
			"card_id39" : idCard[38],
			"card_nb39" : nbCard[38],
			"card_id40" : idCard[39],
			"card_nb40" : nbCard[39],
			
		},
	)
	
}	

func main() {
    router := gin.Default()
    router.GET("/cardlist/:id", getDeckList)
    router.Run("localhost:8080")
	

}