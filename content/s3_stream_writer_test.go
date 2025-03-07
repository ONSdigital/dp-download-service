package content

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/ONSdigital/dp-download-service/content/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testS3Path = "wibble/1234567890.csv"
	testErr    = errors.New("bork")
)

type StubWriter struct {
	data []byte
}

func (w *StubWriter) Write(p []byte) (n int, err error) {
	w.data = p
	return len(w.data), nil
}

func TestStreamWriter_WriteContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("should return expected error if s3 client Get returns an error", t, func() {
		w := writerNeverInvoked(ctrl)
		s3Cli, expectedErr := s3ClientGetReturnsError(ctrl, testS3Path)

		s := &S3StreamWriter{
			S3Client: s3Cli,
		}

		err := s.StreamAndWrite(nil, testS3Path, w)

		So(errors.Is(err, expectedErr), ShouldBeTrue)
	})

	Convey("should return expected error if s3reader returns an error", t, func() {
		w := writerNeverInvoked(ctrl)
		s3ReadCloser, expectedErr := s3ReadCloserErroringOnRead(ctrl)
		s3Cli := s3ClientGetReturnsReader(ctrl, testS3Path, s3ReadCloser)

		s := &S3StreamWriter{
			S3Client: s3Cli,
		}

		err := s.StreamAndWrite(nil, testS3Path, w)

		So(errors.Is(err, expectedErr), ShouldBeTrue)
	})

	Convey("should return expected error if writer.write returns an error", t, func() {
		w, expectedErr := writerReturningErrorOnWrite(ctrl)
		s3ReadCloser := io.NopCloser(strings.NewReader("1, 2, 3, 4"))
		s3Cli := s3ClientGetReturnsReader(ctrl, testS3Path, s3ReadCloser)

		s := &S3StreamWriter{
			S3Client: s3Cli,
		}

		err := s.StreamAndWrite(nil, testS3Path, w)

		So(errors.Is(err, expectedErr), ShouldBeTrue)
	})

	Convey("should successfully write bytes from s3Reader to the provided writer", t, func() {
		readCloser := io.NopCloser(strings.NewReader("1, 2, 3, 4"))
		writer := &StubWriter{}
		s3Cli := s3ClientGetReturnsReader(ctrl, testS3Path, readCloser)

		s := &S3StreamWriter{
			S3Client: s3Cli,
		}

		err := s.StreamAndWrite(nil, testS3Path, writer)

		So(err, ShouldBeNil)
		So(writer.data, ShouldResemble, []byte("1, 2, 3, 4"))
	})

	Convey("should successfully write bytes from s3Reader to the provided writer", t, func() {
		readCloser := io.NopCloser(strings.NewReader("1, 2, 3, 4"))
		writer := &StubWriter{}
		s3Cli := s3ClientGetReturnsReader(ctrl, testS3Path, readCloser)

		s := &S3StreamWriter{
			S3Client: s3Cli,
		}

		err := s.StreamAndWrite(nil, testS3Path, writer)

		So(err, ShouldBeNil)
		So(writer.data, ShouldResemble, []byte("1, 2, 3, 4"))
	})

}

func s3ClientGetReturnsError(ctrl *gomock.Controller, key string) (*mocks.MockS3Client, error) {
	cli := mocks.NewMockS3Client(ctrl)
	cli.EXPECT().Get(gomock.Any(), key).Times(1).Return(nil, nil, testErr)
	return cli, testErr
}

func s3ClientGetReturnsReader(ctrl *gomock.Controller, key string, r S3ReadCloser) *mocks.MockS3Client {
	cli := mocks.NewMockS3Client(ctrl)
	cli.EXPECT().Get(gomock.Any(), key).Times(1).Return(r, nil, nil)
	return cli
}

func s3ReadCloserErroringOnRead(ctrl *gomock.Controller) (*mocks.MockS3ReadCloser, error) {
	r := mocks.NewMockS3ReadCloser(ctrl)
	r.EXPECT().Read(gomock.Any()).MinTimes(1).Return(0, testErr)
	r.EXPECT().Close().Times(1)
	return r, testErr
}

func writerNeverInvoked(ctrl *gomock.Controller) *mocks.MockWriter {
	w := mocks.NewMockWriter(ctrl)
	w.EXPECT().Write(gomock.Any()).Times(0).Return(0, nil)
	return w
}

func writerReturningErrorOnWrite(ctrl *gomock.Controller) (*mocks.MockWriter, error) {
	w := mocks.NewMockWriter(ctrl)
	w.EXPECT().Write(gomock.Any()).Times(1).Return(0, testErr)
	return w, testErr
}
