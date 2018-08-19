package dingtalk

import (
	"bytes"
	"encoding/json"
	"fmt"
	// "io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	// "path"
	"sync/atomic"

	"github.com/influxdata/kapacitor/alert"
	"github.com/influxdata/kapacitor/keyvalue"
	// "github.com/pkg/errors"
)

const DefaultDingtalkURL = "https://oapi.dingtalk.com/robot/send"

type Diagnostic interface {
	WithContext(ctx ...keyvalue.T) Diagnostic
	Error(msg string, err error)
}

type Service struct {
	configValue atomic.Value
	diag        Diagnostic
}

func NewService(c Config, d Diagnostic) *Service {
	s := &Service{
		diag: d,
	}
	s.configValue.Store(c)
	return s
}

func (s *Service) Open() error {
	return nil
}

func (s *Service) Close() error {
	return nil
}

func (s *Service) config() Config {
	return s.configValue.Load().(Config)
}

func (s *Service) Update(newConfig []interface{}) error {
	if l := len(newConfig); l != 1 {
		return fmt.Errorf("expected only one new config object, got %d", l)
	}
	if c, ok := newConfig[0].(Config); !ok {
		return fmt.Errorf("expected config object to be of type %T, got %T", c, newConfig[0])
	} else {
		s.configValue.Store(c)
	}
	return nil
}

type testOptions struct {
	Message     string `json:"message"`
	AccessToken string `json:"access_token"`
}

func (s *Service) TestOptions() interface{} {
	c := s.config()
	return &testOptions{
		Message:     "test dingtalk message",
		AccessToken: c.AccessToken,
	}
}

func (s *Service) Test(options interface{}) error {
	o, ok := options.(*testOptions)
	if !ok {
		return fmt.Errorf("unexpected options type %T", options)
	}
	return s.Send(o.Message)
}

type Message struct {
	Msgtype string      `json:"msgtype"`
	Text    MessageText `json:"text"`
}

type MessageText struct {
	Content string `json:"content"`
}

type Response struct {
	ErrMsg  string `json:"errmsg"`
	ErrCode int    `json:"errcode"`
}

func (s *Service) Send(message string) error {
	c := s.config()
	accessToken := c.AccessToken
	// accessToken = "25399487955846cad9176d62d1851ae29fca7e2735ce9411a975aacdb2f5c957"
	// baseUrl := "https://oapi.dingtalk.com/robot/send"
	// baseUrl := "https://oapi.dingtalk.com/robot/send?access_token=25399487955846cad9176d62d1851ae29fca7e2735ce9411a975aacdb2f5c957"

	v := url.Values{}
	v.Add("access_token", accessToken)
	sendUrl := fmt.Sprintf("%s?%s", DefaultDingtalkURL, v.Encode())
	// log.Println(sendUrl)

	mt := MessageText{
		Content: message,
	}
	ml := Message{
		Msgtype: "text",
		Text:    mt,
	}

	body, err := json.Marshal(ml)

	req, err := http.NewRequest("POST", sendUrl, bytes.NewReader(body))

	if err != nil {
		log.Printf("failed to create POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("failed to POST alert data: %v", err)
	}
	defer resp.Body.Close()

	response, err := ioutil.ReadAll(resp.Body)
	log.Println(string(response))
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		res := &Response{}

		err = json.Unmarshal(body, res)

		if err != nil {
			return fmt.Errorf("failed to understand Dingtalk response (err: %s). code: %d content: %s", err.Error(), resp.StatusCode, string(body))
		}
		return fmt.Errorf("sendMessage error (%d) description: %s", res.ErrCode, res.ErrMsg)
	}
	return nil
}

// func (s *Service) Alert(message string) error {
// 	url, post, err := s.preparePost(message)
// 	if err != nil {
// 		return err
// 	}

// 	resp, err := http.Post(url, "application/json", post)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()
// 	if resp.StatusCode != http.StatusOK {
// 		body, err := ioutil.ReadAll(resp.Body)
// 		if err != nil {
// 			return err
// 		}
// 		type response struct {
// 			ErrMsg  string `json:"errmsg"`
// 			ErrCode int    `json:"errcode"`
// 		}
// 		res := &response{}

// 		err = json.Unmarshal(body, res)

// 		if err != nil {
// 			return fmt.Errorf("failed to understand Dingtalk response (err: %s). code: %d content: %s", err.Error(), resp.StatusCode, string(body))
// 		}
// 		return fmt.Errorf("sendMessage error (%d) description: %s", res.ErrCode, res.ErrMsg)

// 	}
// 	return nil
// }

// func (s *Service) preparePost(message string) (string, io.Reader, error) {
// 	c := s.config()
// 	if !c.Enabled {
// 		return "", nil, errors.New("service is not enabled")
// 	}
// 	postData := make(map[string]interface{})
// 	postData["text"] = message

// 	var post bytes.Buffer
// 	enc := json.NewEncoder(&post)
// 	err := enc.Encode(postData)
// 	if err != nil {
// 		return "", nil, err
// 	}

// 	u, err := url.Parse(DefaultDingtalkURL)
// 	if err != nil {
// 		return "", nil, errors.Wrap(err, "invalid URL")
// 	}
// 	u.Path = path.Join(u.Path+c.AccessToken, "sendMessage")
// 	return u.String(), &post, nil
// }

type HandlerConfig struct {
	AccessToken string `mapstructure:"access-token"`
}

type handler struct {
	s    *Service
	c    HandlerConfig
	diag Diagnostic
}

func (s *Service) Handler(c HandlerConfig, ctx ...keyvalue.T) alert.Handler {
	return &handler{
		s:    s,
		c:    c,
		diag: s.diag.WithContext(ctx...),
	}
}

func (h *handler) Handle(event alert.Event) {
	if err := h.s.Send(event.State.Message); err != nil {
		h.diag.Error("failed to send event to Dingtalk", err)
	}
}
