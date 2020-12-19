package bin

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"io/ioutil"
	"penti/model"
	"penti/spider"
	"penti/utils"
	"time"
)

func FetchList() {
	fmt.Println("fetching list")
	s := spider.New()

	var urlList []string
	for i := 1; i < 2; i++ {
		fmt.Println(fmt.Sprintf("page: %d", i))
		urlList = append(urlList, s.FetchPageList(i)...)
	}

	for index, url := range urlList {
		fmt.Println(fmt.Sprintf("index: %d", index))
		articleModel := s.FetchArticle(url)
		if articleModel.FullTitle != "" {
			fmt.Println(articleModel.FullTitle)
			model.Db.Create(&articleModel)
			articleStruct := utils.Model2Article(articleModel)

			htmlBuffer := utils.RenderHtml(articleStruct)
			file := fmt.Sprintf("%s/%s", utils.GetSaveDir(articleModel), utils.GetSaveName(articleModel))
			ioutil.WriteFile(file, htmlBuffer.Bytes(), 0777)
			fmt.Println("success")
		}

		fmt.Println("sleeping ...")
		time.Sleep(2 * time.Second)
	}
}

func FetchLatestArticle() {
	fmt.Println("Fetching latest article...")
	nowDateStr := time.Now().Format("20060102")
	fmt.Println("Date: " + nowDateStr)
	latest, _ := model.Rdb.Get(model.Ctx, "latest").Result()
	if latest != nowDateStr {
		s := spider.New()
		url, dateStr := s.FetchLatestArticleUrl()
		if dateStr == nowDateStr {
			articleModel := s.FetchArticle(url)
			model.Rdb.Set(model.Ctx, "latest", dateStr, 0)
			err := model.Db.Create(&articleModel).Error
			if err != nil {
				model.Rdb.ZAdd(model.Ctx, "articleList", &redis.Z{
					Score: utils.DateToFloat64(articleModel.Date),
					Member: articleModel.FullTitle,
				})

				articleStruct := utils.Model2Article(articleModel)
				htmlBuffer := utils.RenderHtml(articleStruct)
				file := fmt.Sprintf("%s/%s", utils.GetSaveDir(articleModel), utils.GetSaveName(articleModel))
				ioutil.WriteFile(file, htmlBuffer.Bytes(), 0777)

				fmt.Println(nowDateStr)
			}
		}
	}
}
