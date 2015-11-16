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
	"strconv"
	"strings"

	"github.com/pa001024/reflex/util/ascgen"
)

const (
	SITE_BASE        = "http://www.vjshi.com"
	DEFAULT_PASSWORD = "vjscrapy99"
	DEFAULT_UA       = "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.86 Safari/537.36"
)

// VJshi batch op
func main() {
	isRegist := flag.Bool("r", false, "isRegist: -r require -u -p -q otherwise -u -p")
	isBatch := flag.Bool("b", false, "isBatch: -b")
	u := flag.String("u", "", "username")
	p := flag.String("p", DEFAULT_PASSWORD, "password")
	q := flag.String("q", "", "qq number")
	flag.Parse()
	if *isBatch {
		for i := 1; i <= 99; i++ {
			uname := fmt.Sprintf("vjqq12011158%02d", i)
			v := NewUser(uname, DEFAULT_PASSWORD, "")
			v.LoginAndSign()
		}
	} else {
		v := NewUser(*u, *p, *q)
		if *isRegist {
			err := v.Regist()
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf("注册成功: 账号: %s 密码: %s QQ: %s\n", *u, *p, *q)
			}
		} else {
			v.LoginAndSign()
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
	req, _ := http.NewRequest("POST", SITE_BASE+api, strings.NewReader(param.Encode()))
	req.Header.Set("User-Agent", DEFAULT_UA)
	req.Header.Set("Referer", "http://www.vjshi.com/Passport/usercenter/")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	return this.c.Do(req)
}
func (this *User) getCapture() (res *http.Response, err error) {
	req, _ := http.NewRequest("GET", SITE_BASE+"/Base/verify/t/5122223123", nil)
	req.Header.Set("User-Agent", DEFAULT_UA)
	req.Header.Set("Referer", "http://www.vjshi.com/Passport/usercenter/")
	return this.c.Do(req)
}
func (this *User) Regist() (err error) {
	var err_msg = []string{"注册成功", "当前禁止注册新用户", "账号需要激活", "此用户名已被系统禁止注册", "用户名不合法", "包含不允许注册的词语", "用户名已经存在", "QQ格式有误", "Email不允许注册", "该Email已经被注册", "未定义错误", "", "验证码错误,请检查", "每个IP只能注册6个用户,请输入验证码注册", "", "验证码不能为空,请输入"}
	var verify = ""
ret:
	res, err := this.post("/Passport/doreg/", url.Values{
		"userName": {this.Username},
		"password": {this.Password},
		"email":    {this.QQ + "@qq.com"},
		"verify":   {verify},
	})
	if err != nil {
		return
	}
	rst, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	var dst map[string]string
	json.Unmarshal(rst, &dst)
	code, _ := strconv.Atoi(dst["code"])
	if code == 2 {
		return errors.New(err_msg[code] + " /Passport/activeuser/auth/" + dst["auth"])
	}
	if code == 12 || code == 15 {
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
	if code != 0 {
		return errors.New(err_msg[code])
	}
	return
}

func (this *User) Login() (err error) {
	var err_msg = []string{"登录成功", "账号需要激活", "用户名不存在", "用户名或密码错误", "未知错误", "你的帐户已被禁用", "", "", "连续登陆错误3次，请稍后再试", "验证码错误,请检查"}

	res, err := this.post("/Passport/dologin_ajax/", url.Values{
		"userName": {this.Username},
		"password": {this.Password},
		"verify":   {},
	})

	rst, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	var dst map[string]string
	json.Unmarshal(rst, &dst)
	i, _ := strconv.Atoi(dst["code"])
	if i == 10 {
		return errors.New(dst["info"])
	}
	if i == 1002 {
		return errors.New("账号审核中")
	}
	if i != 0 {
		return errors.New(err_msg[i])
	}
	return
}

func (this *User) Sign() (err error) {
	res, err := this.post("/Passport/daysign/", url.Values{})
	rst, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	res.Body.Close()
	var obj map[string]string
	json.Unmarshal(rst, &obj)
	if obj["code"] == "success" {
		return
	} else {
		return errors.New(obj["info"])
	}
}

func (this *User) LoginAndSign() {
	err := this.Login()
	if err != nil {
		fmt.Println(err)
	} else {
		err = this.Sign()
		if err != nil {
			fmt.Printf("签到失败: 账号: %s \n %s", this.Username, err)
		} else {
			fmt.Printf("签到成功: 账号: %s \n", this.Username)
		}
	}
}
