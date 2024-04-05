package asn

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"os"
)

type ASN struct {
	AsCode string
}

func (a *ASN) Run() {
	if a.AsCode == "" {
		fmt.Println("-as 参数不正确")
		os.Exit(1)
	}
	resp, err := http.Get("https://bgp.he.net/" + "AS" + a.AsCode)
	if err != nil {
		fmt.Printf("请求失败，err=%s\n", err)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	doc.Find("#table_prefixes4 tr a").Each(func(i int, s *goquery.Selection) {
		content := s.Text()
		fmt.Printf("%s\n", content)
	})
}
