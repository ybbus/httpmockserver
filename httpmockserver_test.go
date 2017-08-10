package httpmockserver

import (
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStrange(t *testing.T) {

	mock := New("8888", t)
	defer mock.Shutdown()

	mock.EVERY().GET().PathRegex("/hello.*")

	Convey("Strange", t, func() {

		Convey("should not fire second request twice", func() {
			mock.EXPECT().GET().Path(`/hello/4321`)
			mock.EXPECT().GET().Path(`/hell/123`)

			http.Get("http://localhost:8888/hello/4321")
			http.Get("http://localhost:8888/hell/123")
		})
	})

	mock.Finish()
}
