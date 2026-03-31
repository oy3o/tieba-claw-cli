package api

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"time"
)

type Client struct {
	resty *resty.Client
	token string
}

func NewClient(token string) *Client {
	c := resty.New()
	c.SetBaseURL("https://tieba.baidu.com")
	c.SetHeader("Authorization", token)
	c.SetTimeout(10 * time.Second)

	return &Client{
		resty: c,
		token: token,
	}
}

// Common response structures
type BaseResponse struct {
	ErrNo  int    `json:"errno"`
	ErrMsg string `json:"errmsg"`
}

type GetThreadResponse struct {
	ErrorCode int `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
	Data      struct {
		ThreadList []Thread `json:"thread_list"`
	} `json:"data"`
}

type Thread struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Author   struct {
		Name string `json:"name"`
	} `json:"author"`
	ReplyNum int `json:"reply_num"`
	AgreeNum int `json:"agree_num"`
}

type PageClawResponse struct {
	ErrorCode int `json:"error_code"`
	FirstFloor struct {
		ID      int64  `json:"id"`
		Title   string `json:"title"`
		Content []struct {
			Type int    `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"first_floor"`
	PostList []Post `json:"post_list"`
}

type Post struct {
	ID      int64 `json:"id"`
	Content []struct {
		Type int    `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	SubPostList struct {
		SubPostList []Post `json:"sub_post_list"`
	} `json:"sub_post_list"`
}

type ReplyMeResponse struct {
	No    int    `json:"no"`
	Error string `json:"error"`
	Data  struct {
		ReplyList []struct {
			ThreadID     int64  `json:"thread_id"`
			PostID       int64  `json:"post_id"`
			Title        string `json:"title"`
			Unread       int    `json:"unread"`
			Content      string `json:"content"`
			QuoteContent string `json:"quote_content"`
		} `json:"reply_list"`
	} `json:"data"`
}

type AddThreadResponse struct {
	BaseResponse
	Data struct {
		ThreadID int64 `json:"thread_id"`
		PostID   int64 `json:"post_id"`
	} `json:"data"`
}

type NestedFloorResponse struct {
	ErrorCode int `json:"error_code"`
	Data      struct {
		PostList []Post `json:"post_list"`
	} `json:"data"`
}

// ListThreads gets the forum page threads
func (c *Client) ListThreads(sortType int) (*GetThreadResponse, error) {
	var res GetThreadResponse
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8").
		SetQueryParam("sort_type", fmt.Sprintf("%d", sortType)).
		SetResult(&res).
		Get("/c/f/frs/page_claw")

	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("http error: %s", resp.Status())
	}
	return &res, nil
}

// GetThreadDetails gets the posts in a thread
func (c *Client) GetThreadDetails(threadID int64, pn int) (*PageClawResponse, error) {
	var res PageClawResponse
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8").
		SetQueryParams(map[string]string{
			"kz": fmt.Sprintf("%d", threadID),
			"pn": fmt.Sprintf("%d", pn),
			"r":  "0",
		}).
		SetResult(&res).
		Get("/c/f/pb/page_claw")

	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("http error: %s", resp.Status())
	}
	return &res, nil
}

// AddPost replies to a thread or post
func (c *Client) AddPost(content string, threadID int64, postID int64) (*BaseResponse, error) {
	payload := map[string]interface{}{
		"content": content,
	}
	if threadID != 0 {
		payload["thread_id"] = threadID
	}
	if postID != 0 {
		payload["post_id"] = postID
	}

	var res struct {
		BaseResponse
		Data interface{} `json:"data"`
	}
	_, err := c.resty.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		SetResult(&res).
		Post("/c/c/claw/addPost")

	if err != nil {
		return nil, err
	}
	return &res.BaseResponse, nil
}

// AddThread creates a new thread
func (c *Client) AddThread(title, content string, tabID int, tabName string) (*AddThreadResponse, error) {
	payload := map[string]interface{}{
		"title": title,
		"content": []map[string]string{
			{"type": "text", "content": content},
		},
	}
	if tabID != 0 {
		payload["tab_id"] = tabID
	}
	if tabName != "" {
		payload["tab_name"] = tabName
	}

	var res AddThreadResponse
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		SetResult(&res).
		Post("/c/c/claw/addThread")

	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("http error: %s", resp.Status())
	}
	return &res, nil
}

// OpAgree likes/unlikes a thread or post
func (c *Client) OpAgree(threadID, postID int64, objType, opType int) (*BaseResponse, error) {
	payload := map[string]interface{}{
		"thread_id": threadID,
		"obj_type":  objType,
		"op_type":   opType,
	}
	if postID != 0 {
		payload["post_id"] = postID
	}

	var res BaseResponse
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		SetResult(&res).
		Post("/c/c/claw/opAgree")

	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("http error: %s", resp.Status())
	}
	return &res, nil
}

// ModifyName changes the user's nickname
func (c *Client) ModifyName(name string) (*BaseResponse, error) {
	payload := map[string]string{"name": name}
	var res BaseResponse
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		SetResult(&res).
		Post("/c/c/claw/modifyName")

	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("http error: %s", resp.Status())
	}
	return &res, nil
}

// ReplyMe fetches incoming replies
func (c *Client) ReplyMe(pn int) (*ReplyMeResponse, error) {
	var res ReplyMeResponse
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8").
		SetQueryParam("pn", fmt.Sprintf("%d", pn)).
		SetResult(&res).
		Get("/mo/q/claw/replyme")

	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("http error: %s", resp.Status())
	}
	return &res, nil
}

// DelThread deletes a thread
func (c *Client) DelThread(threadID int64) (*BaseResponse, error) {
	payload := map[string]int64{"thread_id": threadID}
	var res BaseResponse
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		SetResult(&res).
		Post("/c/c/claw/delThread")

	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("http error: %s", resp.Status())
	}
	return &res, nil
}

// DelPost deletes a post/reply
func (c *Client) DelPost(postID int64) (*BaseResponse, error) {
	payload := map[string]int64{"post_id": postID}
	var res BaseResponse
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		SetResult(&res).
		Post("/c/c/claw/delPost")

	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("http error: %s", resp.Status())
	}
	return &res, nil
}

// GetNestedFloor gets replies to a specific post
func (c *Client) GetNestedFloor(threadID, postID int64) (*NestedFloorResponse, error) {
	var res NestedFloorResponse
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8").
		SetQueryParams(map[string]string{
			"thread_id": fmt.Sprintf("%d", threadID),
			"post_id":   fmt.Sprintf("%d", postID),
		}).
		SetResult(&res).
		Get("/c/f/pb/nestedFloor_claw")

	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("http error: %s", resp.Status())
	}
	return &res, nil
}

// DownloadFile downloads a file from URL to dest
func (c *Client) DownloadFile(url, dest string) (*resty.Response, error) {
	return c.resty.R().
		SetOutput(dest).
		Get(url)
}
