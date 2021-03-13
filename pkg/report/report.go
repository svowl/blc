package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/smtp"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"blc/pkg/conf"
	"blc/pkg/crawler"
)

const (
	reportFileDateLayout = "2006-01-02-15-04-05"
	reportDateLayout     = "2006-01-02 15:04:05"
)

// JSONData is report data structure
type JSONData struct {
	TimeElapsed  time.Duration
	TimeFinished time.Time
	TotalLinks   int
	URLs         []string
	Errors       map[string]crawler.ErrorResult
}

// Save saves a report in file
func Save(data *JSONData) (string, error) {
	encoded, err := json.MarshalIndent(*data, "", "    ")
	if err != nil {
		return "", err
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir += "/reports"
	os.MkdirAll(dir, 0755)
	now := time.Now()
	filename := fmt.Sprintf("%s/report-%s.json", dir, now.Format(reportFileDateLayout))

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = f.Write(encoded)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// SaveCSV saves a report as CSV-file
func SaveCSV(data *JSONData) (string, error) {
	encoded, err := csvReport(*data)
	if err != nil {
		return "", err
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir += "/reports"
	os.MkdirAll(dir, 0755)
	now := time.Now()
	filename := fmt.Sprintf("%s/report-%s.csv", dir, now.Format(reportFileDateLayout))

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = f.Write(encoded)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// Send send report to specified email
func Send(cfg conf.SMTP, repData JSONData) error {

	auth := login(cfg.Username, cfg.Password)

	to := []string{cfg.To}

	textBody, err := mailTextBody(repData)
	if err != nil {
		return err
	}

	HTMLBody, err := mailHTMLBody(repData)
	if err != nil {
		return err
	}
	if len(HTMLBody) == 0 {
		return fmt.Errorf("Error of body template parsing")
	}

	csvContent := ""
	if csvBytes, err := csvReport(repData); err == nil && len(csvBytes) > 0 {
		csvContent = "--------=_NextPart_000_0001_01D6F248.DF431190\r\n" +
			"Content-Type: text/csv\r\n" +
			"Content-Disposition: attachment; filename=errors.csv\r\n" +
			"Content-Description: complete errors list\r\n" +
			"\r\n" +
			string(csvBytes) +
			"\r\n"
	}

	msg := []byte("To: <" + to[0] + ">\r\n" +
		"From: <" + cfg.From + ">\r\n" +
		"Reply-To: <" + cfg.ReplyTo + ">\r\n" +
		"Date: " + time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700") + "\r\n" +
		"Message-ID: <" + time.Now().Format("150405.00000") + "@" + cfg.Domain + ">\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Subject: Broken Links Report\r\n" +
		"Content-Type: multipart/alternative; boundary=\"------=_NextPart_000_0001_01D6F248.DF431190\"\r\n" +
		"\r\n" +
		"--------=_NextPart_000_0001_01D6F248.DF431190\r\n" +
		"Content-Type: text/plain; charset=\"utf-8\"\r\n" +
		"Content-Transfer-Encoding: 8bit\r\n" +
		"Content-Disposition: inline\r\n" +
		"\r\n" +
		textBody +
		"\r\n" +
		"--------=_NextPart_000_0001_01D6F248.DF431190\r\n" +
		"Content-Type: text/html; charset=\"utf-8\"\r\n" +
		"Content-Transfer-Encoding: 8bit\r\n" +
		"Content-Disposition: inline\r\n" +
		"\r\n" +
		HTMLBody +
		csvContent +
		"\r\n" +
		"--------=_NextPart_000_0001_01D6F248.DF431190--\r\n")
	err = smtp.SendMail(cfg.Addr, auth, cfg.From, to, msg)
	if err != nil {
		return err
	}
	return nil
}

// mailHTMLBody prepares html body for email
func mailHTMLBody(repData JSONData) (string, error) {
	var b []byte
	buf := bytes.NewBuffer(b)

	tpl := `
<!doctype html>
<html>
	<head>
		<meta http-equiv="content-type" content="text/html; charset=utf-8" />
		<style>
			.tbl { width: 100%; table-layout: fixed; }
			.th { text-align: left; white-space: nowrap; background-color: #444; color: #fff;}
			.ls { padding-left: 10px; }
			.th1,.th4 { width: 35%; }
			.th2 { width: 10%; }
			.th3 { width: 20%; }
		</style>
	</head>
	<body>
		<h1>Broken Links Generator report</h1>
		<b>URLs to scan:</b>
		<ul>
		{{range $url := .URLs}}
		<li>{{$url}}</li>
		{{end}}
		</ul>
		<div>Generated: {{.TimeFinished | formatTime}}
		<br />
		Time taken: {{.TimeElapsed | formatDuration}}
		</div>
		<div style="font-weight: bold;">Total links processed: {{ .TotalLinks }}</div>
		<div style="font-weight: bold; color: #dc3545;">Total errors: {{len .Errors }}</div>
		<br />
		{{if len .Errors}}
		<table border="0" cellspacing="0" cellpadding="3" class="tbl">
			<thead>
				<tr>
					<th class="th th1 ls">URL</th>
					<th class="th th2">HTTP code</th>
					<th class="th th3">Error</th>
					<th class="th th4">Parent URL</th>
				</tr>
			</thead>
			<tbody>
			{{range $url, $err := .Errors}}
				<tr>
					<td>{{$url}}</td>
					<td>{{$err.HTTPStatus}}</td>
					<td>{{$err.Error}}</td>
					<td>{{$err.ParentURL}}</td>
				</tr>
			{{end}}
			</tbody>
		</table>
		{{end}}
	</body>
</html>
`
	t := template.Must(template.New("main").Funcs(template.FuncMap{"formatTime": formatTime, "formatDuration": formatDuration}).Parse(tpl))

	if err := t.Execute(buf, repData); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// formatTime returns time.Time value as a string
func formatTime(t time.Time) string {
	return t.Format("Mon 2 Jan 2006, at 15:04:05 MST")
}

// formatTime returns time.Time value as a string
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%2d hr %02d min %02d sec", h, m, s)
}

// cutString formats long URL as short URL with first n chars and last m chars and '...' between them
// This function is not used at the moment
func cutString(url string) string {
	if len(url) < 100 {
		return url
	}
	var a, b = 50, 20
	if len(url)-b < a+5 {
		return url
	}
	return url[:a] + "..." + url[len(url)-b:len(url)-1]
}

// mailTextBody prepares text body for email
func mailTextBody(repData JSONData) (string, error) {
	var b []byte
	buf := bytes.NewBuffer(b)

	tpl := `
URLs to scan:
{{range $url := .URLs}}
- {{$url}}
{{end}}
Total links processed: {{ .TotalLinks }}
Total errors: {{len .Errors }}

{{if len .Errors}}
{{range $url, $err := .Errors}}
URL: {{$url}}
HTTP code: {{$err.HTTPStatus}}
Error: {{$err.Error}}
Parent URL: {{$err.ParentURL}}
{{end}}
{{end}}
`
	t := template.Must(template.New("main").Parse(tpl))

	if err := t.Execute(buf, repData); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Reports returns list of reports' files
func Reports() ([]string, error) {
	list := make([]string, 0)

	files, err := reportFiles()
	if err != nil {
		return list, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		repDateStr := strings.TrimLeft(file.Name(), "report-")
		repDateStr = strings.TrimRight(repDateStr, ".json")
		repDate, err := time.Parse(reportFileDateLayout, repDateStr)
		if err != nil {
			continue
		}
		list = append(list, repDate.Format(reportDateLayout))
	}
	return list, nil
}

// Report returns JSON-encoded report data by its date
func Report(date string) ([]byte, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	dir += "/reports"
	repDate, err := time.Parse(reportDateLayout, date)
	if err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("%s/report-%s.json", dir, repDate.Format(reportFileDateLayout))
	var result []byte
	result, err = ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CleanReports removes all old reports except 10 most recent ones
func CleanReports(maxReportsToStore int) error {
	files, err := reportFiles()
	if err != nil {
		return err
	}
	if len(files) <= maxReportsToStore {
		return nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	dir += "/reports"
	var cnt int
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		cnt++
		if cnt <= maxReportsToStore {
			continue
		}
		if err := os.Remove(dir + "/" + file.Name()); err != nil {
			return err
		}
	}
	return nil
}

func csvReport(repData JSONData) ([]byte, error) {
	if len(repData.Errors) == 0 {
		return nil, nil
	}
	records := make([][]string, 0, len(repData.Errors)+1)
	records = append(records, []string{
		"URL",
		"HTTP code",
		"Error",
		"Parent URL",
	})
	for u, e := range repData.Errors {
		record := []string{
			u,
			strconv.Itoa(e.HTTPStatus),
			e.Error,
			e.ParentURL,
		}
		records = append(records, record)
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.WriteAll(records)

	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("error writing csv: %v", err)
	}

	return b.Bytes(), nil
}

// reportFiles returns list of all reports as []FileInfo
func reportFiles() ([]os.FileInfo, error) {
	list := make([]os.FileInfo, 0)

	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	dir += "/reports"
	if _, err := os.Stat(dir); err != nil {
		return list, nil
	}
	f, err := os.Open(dir)
	if err != nil {
		return list, nil
	}
	files, err := f.Readdir(0)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		list = append(list, file)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() > list[j].Name() })
	return list, nil
}
