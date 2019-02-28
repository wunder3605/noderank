package noderank

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/awalterschulze/gographviz"
	"github.com/kylelemons/go-gypsy/yaml"
	"io/ioutil"
	"log"
	"net/http"
	url2 "net/url"
	"pagerank"
	"sort"
	"strconv"
	"time"
)

type Response struct {
	Blocks   string `json:"blocks"`
	Duration int    `json:"duration"`
}

type message struct {
	TeeNum     int64    `json:"tee_num"`
	TeeContent []teectx `json:"tee_content"`
}

type teectx struct {
	Attester string  `json:"attester"`
	Attestee string  `json:"attestee"`
	Score    float64 `json:"score"`
}

type rawtxnslice []teectx

var url = "http://localhost:14700"
var addr = "JVSVAFSXWHUIZPFDLORNDMASGNXWFGZFMXGLCJQGFWFEZWWOA9KYSPHCLZHFBCOHMNCCBAGNACPIGHVYX"

var (
	file = flag.String("file", "config.yaml", "IOTA CONFIGURATION")
)

func AddAttestationInfo(info []string) {
	raw := new(teectx)
	raw.Attester = info[0]
	raw.Attestee = info[1]
	num, err := strconv.ParseUint(info[2], 10, 64)
	raw.Score = float64(num)
	m := new(message)
	m.TeeNum = 1
	m.TeeContent = []teectx{*raw}
	ms, err := json.Marshal(m)
	if err != nil {
		log.Panic(err)
	}

	addr1 := getConfigParam("addr")
	if addr1 == "" {
		log.Fatal(err)
		addr1 = addr
	}

	d := time.Now()
	ds := d.Format("20190227")
	data := "{\"command\":\"storeMessage\",\"address\":" + addr1 + ",\"message\":" + url2.QueryEscape(string(ms[:])) + ",\"tag\":\"" + ds + "TEE\"}"
	fmt.Println("data : " + data)
	r := doPost([]byte(data))
	fmt.Println(r)
}

func GetRank(period string, numRank int64) []teectx {
	data := "{\"command\":\"getBlocksInPeriodStatement\",\"period\":" + period + "}"
	r := doPost([]byte(data))
	var result Response
	err := json.Unmarshal(r, &result)
	if err != nil {
		log.Fatal(err)
		fmt.Println(r)
	}
	fmt.Println(result.Duration)
	fmt.Println(result.Blocks)

	var msgArr []string
	err = json.Unmarshal([]byte(result.Blocks), &msgArr)
	if err != nil {
		log.Panic(err)
	}

	graph := pagerank.NewGraph()

	for _, m2 := range msgArr {
		msgT, err := url2.QueryUnescape(m2)
		if err != nil {
			log.Panicln(err)
		}
		var msg message
		err = json.Unmarshal([]byte(msgT), &msg)
		if err != nil {
			log.Panic(err)
		}

		rArr := msg.TeeContent
		for _, r := range rArr {
			graph.Link(r.Attester, r.Attestee, r.Score)
		}
	}

	var rst []teectx
	graph.Rank(0.85, 0.0001, func(attestee string, score float64) {
		fmt.Println("attestee ", attestee, " has a score of", score)
		tee := teectx{"", attestee, score}
		rst = append(rst, tee)
	})
	sort.Sort(rawtxnslice(rst))
	fmt.Println(rst[0:numRank])
	return rst[0:numRank]
}

func PrintHCGraph(period string) {
	data := "{\"command\":\"getBlocksInPeriodStatement\",\"period\":" + period + "}"
	r := doPost([]byte(data))
	var result Response
	err := json.Unmarshal(r, &result)
	if err != nil {
		log.Fatal(err)
		fmt.Println(r)
	}
	fmt.Println(result.Duration)
	fmt.Println(result.Blocks)

	var msgArr []string
	err = json.Unmarshal([]byte(result.Blocks), &msgArr)
	if err != nil {
		log.Panic(err)
	}

	graph := gographviz.NewGraph()

	for _, m2 := range msgArr {
		msgT, err := url2.QueryUnescape(m2)
		if err != nil {
			log.Panicln(err)
		}
		fmt.Println("message : " + msgT)
		var msg message
		err = json.Unmarshal([]byte(msgT), &msg)
		if err != nil {
			log.Panic(err)
		}

		rArr := msg.TeeContent
		for _, r := range rArr {
			//score := strconv.FormatUint(uint64(r.Score), 10) // TODO add this score info
			graph.AddNode("G", r.Attestee, nil)
			graph.AddNode("G", r.Attester, nil)
			graph.AddEdge(r.Attester, r.Attestee, true, nil)
			if err != nil {
				log.Panic(err)
			}
		}
	}

	output := graph.String()
	fmt.Println(output)
}

func doPost(d []byte) []byte {
	uri := getConfigParam("url")
	if uri == "" {
		uri = url
	}
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(d))
	if err != nil {
		// error
		log.Panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-IOTA-API-Version", "1")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Panic(err)
	}

	defer res.Body.Close()
	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Panic(err)
	}
	return r
}

func getConfigParam(p string) string {
	config, err := yaml.ReadFile(*file)
	if err != nil {
		log.Panicln(err)
	}
	result, err := config.Get(p)
	if err != nil {
		log.Panicln(err)
	}
	return result
}

func (r rawtxnslice) Len() int {
	return len(r)
}

func (r rawtxnslice) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r rawtxnslice) Less(i, j int) bool {
	return r[j].Score < r[i].Score
}
