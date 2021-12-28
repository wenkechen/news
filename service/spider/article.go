package spider

import (
	"bytes"
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/henrylee2cn/mahonia"
	"gorm.io/datatypes"
	"html/template"
	"io"
	"log"
	"net/http"
	"news/model"
	"regexp"
	"strings"
	"time"
)

const ListSelector = "table.oblog_t_1.ke-zeroborder ul li"

var spiderCli *Spider

type Spider struct {
}

type Paragraph struct {
	Title  string `json:"title"`
	Bodies []Body `json:"body"`
}

type Body struct {
	Type    string        `json:"type"`
	Content template.HTML `json:"content"`
}

var client = http.DefaultClient

func New() *Spider {
	if spiderCli == nil {
		log.Println("实例化spider")
		spiderCli = new(Spider)
		log.Println("实例化spider成功")
	}
	return spiderCli
}

func (a *Spider) FetchPageList() (urlList []string) {
	doc := a.getRequestReader("https://www.dapenti.com/blog/blog.asp?subjectid=70&name=xilei")
	if doc == nil {
		return nil
	}

	doc.Find(ListSelector).Each(func(i int, selection *goquery.Selection) {
		href, exist := selection.Find("a").Attr("href")
		if exist {
			urlList = append(urlList, "https://www.dapenti.com/blog/"+href)
		}
	})
	return
}

func (a Spider) FetchLatestArticleUrl() (url, dateStr string) {
	log.Println("爬取首页")
	doc := a.getRequestReader("https://www.dapenti.com/blog/index.asp")
	if doc == nil {
		return
	}
	log.Println("请求成功")

	aNode := doc.Find(".box3 .title_down").Last().Find("li").First().Find("a")

	href, exist := aNode.Attr("href")
	if exist {
		log.Println("拼接最新文章链接")
		url = "https://www.dapenti.com/blog/" + href
		title := aNode.Text()
		flysnowRegexp := regexp.MustCompile(`\d+`)
		dateStr = flysnowRegexp.FindString(title)
		log.Println("获取文章日期成功")
	}
	log.Println("爬取首页成功")

	return
}

func (a Spider) FetchArticle(url string) (article model.Article) {
	doc := a.getRequestReader(url)
	log.Println("获取文章doc成功")

	if doc == nil {
		return model.Article{}
	}
	article.Url = url

	article.FullTitle = strings.ReplaceAll(doc.Find(`.style1 a`).Last().Text(), "喷嚏图卦", "")

	flysnowRegexp := regexp.MustCompile(`\d+`)
	dateStr := flysnowRegexp.FindString(article.FullTitle)
	dateTime, _ := time.ParseInLocation("20060102", dateStr, time.Local)
	article.Date = datatypes.Date(dateTime)

	titleArr := strings.Split(article.FullTitle, "】")
	if len(titleArr) != 2 {
		return model.Article{}
	}
	article.RealTitle = titleArr[1]

	begin := false
	paragraph := 0
	var paragraphs []*Paragraph
	doc.Find("table.ke-zeroborder p").EachWithBreak(func(i int, selection *goquery.Selection) bool {
		reg := regexp.MustCompile(`【\d+】`)
		titleText := reg.FindString(strings.TrimSpace(selection.Text()))
		if len(titleText) > 0 {
			begin = true
			paragraph++
			paragraphs = append(paragraphs, &Paragraph{
				Title: strings.TrimSpace(selection.Text()),
			})
		} else {
			if begin == true {
				paragraphMain := paragraphs[len(paragraphs)-1]

				var body = Body{}
				if selection.Find("img").Length() > 0 {
					src, _ := selection.Find("img").Attr("src")
					body.Type = "img"
					body.Content = template.HTML(src)
				} else if len(selection.Text()) > 0 {
					if strings.Contains(selection.Text(), "来源：喷嚏网") ||
						strings.Contains(selection.Text(), "item.taobao") ||
						strings.TrimSpace(selection.Text()) == "广告" ||
						strings.Contains(selection.Text(), "本期图卦由") {
						return false
					}
					body.Type = "text"
					contentHtml, _ := selection.Html()
					body.Content = template.HTML(strings.TrimSpace(contentHtml))
				}

				if len(body.Content) > 0 {
					paragraphMain.Bodies = append(paragraphMain.Bodies, body)
				}
			}
		}

		return true
	})
	paragraphsBytes, _ := json.Marshal(paragraphs)
	article.Paragraphs = paragraphsBytes

	return article
}

func (a Spider) getRequestReader(url string) *goquery.Document {
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	enc := mahonia.NewDecoder("gb2312")
	body := enc.NewReader(resp.Body)

	var bodyBytes bytes.Buffer
	_, err = io.Copy(&bodyBytes, body)
	//bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		log.Println("读取body失败")
		return nil
	}
	pageStr := bodyBytes.String()
	reg := regexp.MustCompile(`<hr>广告.*<hr><br>`)
	adStr := reg.FindString(pageStr)

	pureStr := strings.ReplaceAll(pageStr, adStr, "")

	pureReader := strings.NewReader(pureStr)

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(pureReader)
	if err != nil {
		return nil
	}

	return doc
}
