package dto

type RequestBody struct {
	CppCode   string   `json:"cpp_code"`
	TestCases []string `json:"test_cases"`
}

type ResponseBody struct {
	Results []TestResult `json:"results"`
}

type TestResult struct {
	Result string `json:"result"`
	Time   string `json:"time"`
	Memory string `json:"memory"`
}
