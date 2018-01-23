package instagram_web_client

import (
	"net/http"
	"fmt"
	"net/http/cookiejar"
	"golang.org/x/net/publicsuffix"
	"net/url"
	"strings"
	"io/ioutil"
	"github.com/pkg/errors"
	"encoding/json"
)

type InstagramWebClient struct {
	Client    *http.Client
	CSRFToken string
	Header    map[string]string
}

var INSTAGRAM_ROOT = url.URL{
	Host:   "www.instagram.com",
	Scheme: "https",
}

var GRAPHQL_ROOT = url.URL{
	Host:   "www.instagram.com",
	Path:   "/graphql/query/",
	Scheme: "https",
}

var POPULAR_TAGS = []string{"love", "followback", "instagramers", "socialenvy", "PleaseForgiveMe", "tweegram", "photooftheday", "20likes", "amazing", "smile", "follow4follow", "like4like", "look", "instalike", "igers", "picoftheday", "food", "instadaily", "instafollow", "followme", "girl", "instagood", "bestoftheday", "instacool", "socialenvyco", "follow", "colorful", "style", "swag"}

func Init(password string, username string, cookieString string) (instagramWebClient InstagramWebClient, err error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return
	}
	client := &http.Client{
		Jar: jar,
	}
	instagramWebClient.Client = client
	if cookieString != "" {
		header := http.Header{}
		header.Add("Cookie", cookieString)
		request := http.Request{Header: header}
		instagramWebClient.Client.Jar.SetCookies(&INSTAGRAM_ROOT, request.Cookies())
		for _, cookie := range request.Cookies() {
			if cookie.Name == "csrftoken" {
				instagramWebClient.CSRFToken = cookie.Value
				break
			}
		}
	} else {
		res, err := client.Head("https://www.instagram.com/")
		if err != nil {
			return instagramWebClient, err
		}
		var csrftoken string
		for _, cookie := range res.Cookies() {
			if cookie.Name == "csrftoken" {
				csrftoken = cookie.Value
			}
		}
		instagramWebClient.CSRFToken = csrftoken
		instagramWebClient.Client.Jar.SetCookies(&INSTAGRAM_ROOT, res.Cookies())
		_, err = instagramWebClient.PostLogin(password, username)
		if err != nil {
			return instagramWebClient, err
		}
		return instagramWebClient, err
	}
	return instagramWebClient, err
}

type Login struct {
	Authenticated bool   `json:"authenticated"`
	User          bool   `json:"user"`
	Status        string `json:"status"`
}

func (i InstagramWebClient) PostLogin(password string, username string) (Login, error) {
	params := url.Values{}
	params.Set("password", password)
	params.Set("username", username)
	res, err := i.makeRequest(http.MethodPost, "https://www.instagram.com/accounts/login/ajax/", params)
	if err != nil {
		return Login{}, err
	} else {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return Login{}, err
		} else {
			var data Login
			err := json.Unmarshal(body, &data)
			if err != nil {
				return Login{}, err
			}
			if !data.Authenticated {
				return Login{}, errors.New("Wrong email or password.")
			}
			return data, nil
		}
	}
}

type PostLike struct {
	Status string `json:"status"`
}

func (i InstagramWebClient) PostPostLike(id string) (PostLike, error) {
	res, err := i.makeRequest(http.MethodPost, fmt.Sprintf("https://www.instagram.com/web/likes/%s/like/", id), nil)
	if err != nil {
		return PostLike{}, err
	} else {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return PostLike{}, err
		} else {
			var data PostLike
			err := json.Unmarshal(body, &data)
			if err != nil {
				return PostLike{}, err
			}
			return data, nil
		}
	}
}

type TagFeed struct {
	Data struct {
		Hashtag struct {
			Name string `json:"name"`
			EdgeHashtagToMedia struct {
				PageInfo struct {
					HasNextPage bool   `json:"has_next_page"`
					EndCursor   string `json:"end_cursor"`
				} `json:"page_info"`
				Edges []struct {
					Node struct {
						Id        string `json:"id"`
						Shortcode string `json:"shortcode"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"edge_hashtag_to_media"`
		} `json:"hashtag"`
	} `json:"data"`
}

func (i InstagramWebClient) GetTagFeed(tag string) (TagFeed, error) {
	params := url.Values{}
	params.Set("query_id", "17875800862117404")
	params.Set("tag_name", tag)
	params.Set("first", "10")
	res, err := i.makeRequest(http.MethodGet, GRAPHQL_ROOT.String(), params)
	if err != nil {
		return TagFeed{}, err
	} else {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return TagFeed{}, err
		} else {
			var data TagFeed
			err := json.Unmarshal(body, &data)
			if err != nil {
				return TagFeed{}, err
			}
			return data, nil
		}
	}
}

type UserFollowing struct {
	Data struct {
		User struct {
			EdgeFollow struct {
				Count int64 `json:"count"`
				Edges []struct {
					Node struct {
						Id            string `json:"id"`
						ProfilePicUrl string `json:"profile_pic_url"`
						Username      string `json:"username"`
					} `json:"node"`
				}
				PageInfo struct {
					HasNextPage bool   `json:"has_next_page"`
					EndCursor   string `json:"end_cursor"`
				} `json:"page_info"`
			} `json:"edge_follow"`
		} `json:"user"`
	} `json:"data"`
}

func (i InstagramWebClient) GetUserFollowing() (UserFollowing, error) {
	params := url.Values{}
	params.Set("query_id", "17874545323001329")
	params.Set("id", i.GetUserId().Id)
	params.Set("first", "10")
	res, err := i.makeRequest(http.MethodGet, GRAPHQL_ROOT.String(), params)
	if err != nil {
		return UserFollowing{}, err
	} else {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return UserFollowing{}, err
		} else {
			var data UserFollowing
			err := json.Unmarshal(body, &data)
			if err != nil {
				return UserFollowing{}, err
			}
			return data, nil
		}
	}
}

type UserId struct {
	Id string `json:"id"`
}

func (i InstagramWebClient) GetUserId() (userId UserId) {
	for _, cookie := range i.Client.Jar.Cookies(&INSTAGRAM_ROOT) {
		if cookie.Name == "ds_user_id" {
			userId.Id = cookie.Value
			break
		}
	}
	return
}

func (i InstagramWebClient) CookieString() (cookieString string) {
	var cookies []string
	for _, cookie := range i.Client.Jar.Cookies(&INSTAGRAM_ROOT) {
		cookies = append(cookies, cookie.String())
	}
	return strings.Join(cookies, ";") + ";"
}

func (i *InstagramWebClient) makeRequest(method string, url string, body url.Values) (res *http.Response, err error) {
	var req *http.Request
	if method == http.MethodGet {
		req, err = http.NewRequest(method, url+"?"+body.Encode(), nil)
	}
	if method == http.MethodPost {
		if body != nil {
			req, err = http.NewRequest(method, url, strings.NewReader(body.Encode()))
		} else {
			req, err = http.NewRequest(method, url, nil)
		}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_5) AppleWebKit/601.6.17 (KHTML, like Gecko) Version/9.1.1 Safari/601.6.17")
	req.Header.Set("Acept", "*/*")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("Connection", "close")

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("x-csrftoken", i.CSRFToken)
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("x-instagram-ajax", "1")
	req.Header.Set("Referer", "https://www.instagram.com")
	req.Header.Set("Authority", "www.instagram.com")
	req.Header.Set("Origin", "https://www.instagram.com")

	if err != nil {
		return &http.Response{}, err
	}
	res, err = i.Client.Do(req)
	if err != nil {
		return &http.Response{}, err
	}
	if res.StatusCode != 200 {
		return &http.Response{}, errors.New("HTTPError")
	}

	i.Client.Jar.SetCookies(&INSTAGRAM_ROOT, res.Cookies())
	for _, cookie := range i.Client.Jar.Cookies(&INSTAGRAM_ROOT) {
		if cookie.Name == "csrftoken" {
			i.CSRFToken = cookie.Value
			break
		}
	}
	return res, nil
}
