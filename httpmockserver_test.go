package httpmockserver

import (
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStrange(t *testing.T) {

	mock := New(t)
	url := mock.Url
	defer mock.Shutdown()

	mock.EVERY().GET().PathRegex("/hello.*")

	Convey("Strange", t, func() {

		Convey("should not fire second request twice", func() {
			mock.EXPECT().GET().Path(`/hello/4321`)
			mock.EXPECT().GET().Path(`/hell/123`)

			http.Get(url + "/hello/4321")
			http.Get(url + "/hell/123")
		})
	})

	mock.Finish()
}
