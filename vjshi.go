package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/pa001024/reflex/util"
	"github.com/pa001024/reflex/util/ascgen"
)

const (
	SITE_BASE        = "http://www.vjshi.com"
	PASSPORT_BASE    = "http://passport.vjshi.com"
	DEFAULT_PASSWORD = "vjscrapy99"
	DEFAULT_UA       = "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.86 Safari/537.36"
)

type UserState struct {
	name    string
	success bool
	bp      int
}

// VJshi batch op
func main() {
	isRegist := flag.Bool("r", false, "isRegist: -r require -u -p -q otherwise -u -p")
	isBatch := flag.Bool("b", false, "isBatch: -b")
	prefix := flag.String("prefix", "vjqq12011158", "username prefix")
	u := flag.String("u", "", "username")
	p := flag.String("p", DEFAULT_PASSWORD, "password")
	q := flag.String("q", "", "qq number")
	flag.Parse()
	if *isBatch {
		unamefmt := *prefix + "%02d"
		key_list := make([]string, 0, 99)
		key2user := make(map[string]UserState)
		for i := 1; i <= 99; i++ {
			uname := fmt.Sprintf(unamefmt, i)
			key_list = append(key_list, uname)
			v := NewUser(uname, DEFAULT_PASSWORD, "")
			err := v.LoginAndSign()
			bp := 0
			for i := 0; i < 3; i++ {
				if err != nil {
					err = v.LoginAndSign()
				} else {
					for j := 0; j < 3; j++ {
						if bp == 0 {
							bp, _ = v.GetBounsPoint()
						}
					}
					break
				}
			}
			if err != nil {
				if err.Error() == "您已经签到过了" {
					err = nil
				} else {
					util.ERROR.Log(err)
				}
			}
			key2user[uname] = UserState{uname, err == nil, bp}
		}
		count_fail, count_bp_all := 0, 0
		failed_list := ""
		for _, v := range key_list {
			if !key2user[v].success {
				count_fail++
				failed_list += fmt.Sprintf("vjshi -u %s\n", key2user[v].name)
			}
			count_bp_all += key2user[v].bp
		}
		if count_fail == 0 {
			util.INFO.Logf("%s ~ %s 全部签到完毕\n", fmt.Sprintf(unamefmt, 1), fmt.Sprintf(unamefmt, 99))
		} else {
			util.INFO.Logf("%s ~ %s 部分签到完毕 失败 %d 个\n", fmt.Sprintf(unamefmt, 1), fmt.Sprintf(unamefmt, 99), count_fail)
			fmt.Println("输入以下指令重试:")
			fmt.Println(failed_list)
		}
		util.INFO.Logf("今日共获取 %d 积分 总计 %d 积分 平均每账号 %d 积分\n",
			(99-count_fail)*10,
			count_bp_all,
			count_bp_all/99,
		)
	} else {
		if *u == "" {
			flag.Usage()
			return
		}
		v := NewUser(*u, *p, *q)
		if *isRegist {
			err := v.Register()
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf("注册成功: 账号: %s 密码: %s QQ: %s\n", *u, *p, *q)
			}
		} else {
			err := v.LoginAndSign()
			if err != nil {
				fmt.Println(err)
			} else {
				bp, _ := v.GetBounsPoint()
				fmt.Printf("签到成功: 账号: %s 当前积分: %d", *u, bp)
			}
		}
	}
}

type User struct {
	c        *http.Client
	Username string `json:"username"`
	Password string `json:"pwd"`
	QQ       string `json:"qq"`
}

func NewUser(username, pwd, qq string) (v *User) {
	jar, _ := cookiejar.New(nil)
	v = &User{&http.Client{Jar: jar}, username, pwd, qq}
	return
}

func (this *User) post(api string, param url.Values) (res *http.Response, err error) {
	req, _ := http.NewRequest("POST", PASSPORT_BASE+api, strings.NewReader(param.Encode()))
	req.Header.Set("User-Agent", DEFAULT_UA)
	req.Header.Set("Referer", "http://www.vjshi.com/")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	return this.c.Do(req)
}
func (this *User) getCapture() (res *http.Response, err error) {
	return this.get("/Base/verify/t/5122223123", "")
}
func (this *User) get(api, base string) (res *http.Response, err error) {
	if base == "" {
		base = PASSPORT_BASE
	}
	req, _ := http.NewRequest("GET", base+api, nil)
	req.Header.Set("User-Agent", DEFAULT_UA)
	req.Header.Set("Referer", "http://www.vjshi.com/")
	return this.c.Do(req)
}

// unimpl
func (this *User) Register() (err error) {
	var verify = ""
ret:
	res, err := this.post("/register/ajax", url.Values{
		"username": {this.Username},
		"password": {this.Password},
		"email":    {this.QQ + "@qq.com"}, // 新版可以用任意邮箱注册
		"verify":   {verify},
	})
	if err != nil {
		return
	}
	rst, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	var obj map[string]string
	json.Unmarshal(rst, &obj)
	status := obj["status"]
	if status == "error" {
		res, err = this.getCapture()
		ascgen.ShowFile(os.Stdout, res.Body, ascgen.Console{6, 14, 120}, true, true)
		res.Body.Close()
		fmt.Print("Enter code: ")
		bf := bufio.NewReader(os.Stdin)
		verify, _ = bf.ReadString('\n')
		if len(verify) > 4 {
			verify = verify[:4]
		}
		goto ret
	}
	return errors.New(obj["message"])
}

func (this *User) Login() (err error) {
	res, err := this.get("/login/ajax?"+(url.Values{
		"callback": {"jQuery18309862185863312334_" + util.JsCurrentTime()},
		"username": {this.Username},
		"password": {this.Password},
		"verify":   {},
	}).Encode(), "")
	rst, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	var obj map[string]string
	json.Unmarshal(rst[41:len(rst)-1], &obj)
	if obj["status"] != "success" {
		return errors.New(obj["message"])
	}
	return
}

func (this *User) Sign() (err error) {
	res, err := this.get("/user/daysign?callback=jQuery18300392456746451165_"+util.JsCurrentTime(), "")
	if err != nil {
		return
	}
	rst, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	res.Body.Close()
	var obj map[string]string
	json.Unmarshal(rst[41:len(rst)-1], &obj)
	if obj["status"] != "success" {
		return errors.New(obj["message"])
	}
	return
}

func (this *User) GetBounsPoint() (v int, err error) {
	res, err := this.get("/User/", SITE_BASE)
	if err != nil {
		return
	}
	rst, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	res.Body.Close()
	ex := regexp.MustCompile(`积分：(\d+)</p>`)
	ret := ex.FindStringSubmatch(string(rst))
	if len(ret) > 1 {
		return strconv.Atoi(ret[1])
	}
	return 0, errors.New("未知错误")
}

func (this *User) LoginAndSign() (err error) {
	err = this.Login()
	if err != nil {
		return
	} else {
		err = this.Sign()
		return
	}
	return
}
