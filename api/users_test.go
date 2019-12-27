package api

func (st *ApiTestSuite) TestGetUserInfo() {
	type result struct {
		Mobile string `json:"mobile"`
	}
	res := &result{}
	err := st.Get("/v1/user/info", res)
	st.Require().NoError(err)
	st.Require().Equal(res.Mobile, st.mobile, "mobile error")
}
