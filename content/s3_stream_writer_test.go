package content

import (
	"encoding/hex"
	"errors"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ONSdigital/dp-download-service/content/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testVaultPath  = "wibble"
	testFilename   = "1234567890.csv"
	testS3Filename = "wibble/1234567890.csv"
	testErr        = errors.New("bork")
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

	Convey("should return expected error if filename is empty", t, func() {
		w := writerNeverInvoked(ctrl)
		vaultCli := vaultClientNeverInvoked(ctrl)

		s := &S3StreamWriter{VaultCli: vaultCli}

		err := s.StreamAndWrite(nil, "", w)

		So(err, ShouldEqual, VaultFilenameEmptyErr)
	})

	Convey("should return expected error if vault client read key returns and error", t, func() {
		w := writerNeverInvoked(ctrl)
		vaultCli, expectedErr := vaultClientErrorOnReadKey(ctrl)

		s := &S3StreamWriter{
			VaultPath: testVaultPath,
			VaultCli:  vaultCli,
		}

		err := s.StreamAndWrite(nil, testFilename, w)

		So(err, ShouldEqual, expectedErr)
	})

	Convey("should return expected error if vault client read key returns non hex value", t, func() {
		w := writerNeverInvoked(ctrl)
		vaultCli := vaultClientReturningInvalidHexString(ctrl)

		s := &S3StreamWriter{
			VaultPath: testVaultPath,
			VaultCli:  vaultCli,
		}

		err := s.StreamAndWrite(nil, testFilename, w)

		So(err, ShouldNotBeNil)
	})

	Convey("should return expected error if s3 client GetWithPSK returns an error", t, func() {
		w := writerNeverInvoked(ctrl)
		vaultCli, expectedPSK := vaultClientAndValidKey(ctrl)
		s3Cli, expectedErr := s3ClientGetWithPSKReturnsError(ctrl, testFilename, expectedPSK)

		s := &S3StreamWriter{
			VaultPath: testVaultPath,
			VaultCli:  vaultCli,
			S3Client:  s3Cli,
		}

		err := s.StreamAndWrite(nil, testFilename, w)

		So(err, ShouldEqual, expectedErr)
	})

	Convey("should return expected error if s3reader returns an error", t, func() {
		w := writerNeverInvoked(ctrl)
		vaultCli, expectedPSK := vaultClientAndValidKey(ctrl)
		s3ReadCloser, expectedErr := s3ReadCloserErroringOnRead(ctrl)
		s3Cli := s3ClientGetWithPSKReturnsReader(ctrl, testFilename, expectedPSK, s3ReadCloser)

		s := &S3StreamWriter{
			VaultPath: testVaultPath,
			VaultCli:  vaultCli,
			S3Client:  s3Cli,
		}

		err := s.StreamAndWrite(nil, testFilename, w)

		So(err, ShouldEqual, expectedErr)
	})

	Convey("should return expected error if writer.write returns an error", t, func() {
		w, expectedErr := writerReturningErrorOnWrite(ctrl)
		vaultCli, expectedPSK := vaultClientAndValidKey(ctrl)
		s3ReadCloser := ioutil.NopCloser(strings.NewReader("1, 2, 3, 4"))
		s3Cli := s3ClientGetWithPSKReturnsReader(ctrl, testFilename, expectedPSK, s3ReadCloser)

		s := &S3StreamWriter{
			VaultPath: testVaultPath,
			VaultCli:  vaultCli,
			S3Client:  s3Cli,
		}

		err := s.StreamAndWrite(nil, testFilename, w)

		So(err, ShouldEqual, expectedErr)
	})

	Convey("should successfully write bytes from s3Reader to the provided writer", t, func() {
		readCloser := ioutil.NopCloser(strings.NewReader("1, 2, 3, 4"))
		writer := &StubWriter{}
		vaultCli, expectedPSK := vaultClientAndValidKey(ctrl)
		s3Cli := s3ClientGetWithPSKReturnsReader(ctrl, testFilename, expectedPSK, readCloser)

		s := &S3StreamWriter{
			VaultPath: testVaultPath,
			VaultCli:  vaultCli,
			S3Client:  s3Cli,
		}

		err := s.StreamAndWrite(nil, testFilename, writer)

		So(err, ShouldBeNil)
		So(writer.data, ShouldResemble, []byte("1, 2, 3, 4"))
	})

}

func Test_GetVaultKeyForFile(t *testing.T) {

	Convey("should return error if filename is empty", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		vaultCli := mocks.NewMockVaultClient(ctrl)
		vaultCli.EXPECT().ReadKey(nil, nil).Times(0)

		s := New(vaultCli, "")
		psk, err := s.getVaultKeyForFile("")

		So(psk, ShouldBeNil)
		So(err, ShouldNotBeNil)
		So(err, ShouldEqual, VaultFilenameEmptyErr)
	})

	Convey("should return expected error if vaultClient.Read is unsuccessful", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		vaultCli, expectedErr := vaultClientErrorOnReadKey(ctrl)

		s := New(vaultCli, testVaultPath)
		psk, err := s.getVaultKeyForFile(testFilename)

		So(psk, ShouldBeNil)
		So(err, ShouldNotBeNil)
		So(err, ShouldEqual, expectedErr)
	})

	Convey("should return expected error if the vault key cannot be hex decoded", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		vaultCli := vaultClientReturningInvalidHexString(ctrl)

		s := New(vaultCli, testVaultPath)
		psk, err := s.getVaultKeyForFile(testFilename)

		So(psk, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})

	Convey("should return expected vault key for successful requests", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		vaultCli, keyBytes := vaultClientAndValidKey(ctrl)

		s := New(vaultCli, testVaultPath)
		psk, err := s.getVaultKeyForFile(testFilename)

		So(psk, ShouldResemble, keyBytes)
		So(err, ShouldBeNil)
	})

}

func vaultClientNeverInvoked(ctrl *gomock.Controller) *mocks.MockVaultClient {
	vaultCli := mocks.NewMockVaultClient(ctrl)
	vaultCli.EXPECT().ReadKey(gomock.Any(), gomock.Any()).Times(0)
	return vaultCli
}

func vaultClientErrorOnReadKey(ctrl *gomock.Controller) (*mocks.MockVaultClient, error) {
	vp := filepath.Join(testVaultPath, testFilename)
	vaultCli := mocks.NewMockVaultClient(ctrl)
	vaultCli.EXPECT().ReadKey(vp, vaultKey).Return("", testErr).Times(1)
	return vaultCli, testErr
}

func vaultClientReturningInvalidHexString(ctrl *gomock.Controller) *mocks.MockVaultClient {
	vp := filepath.Join(testVaultPath, testFilename)
	vaultCli := mocks.NewMockVaultClient(ctrl)
	vaultCli.EXPECT().ReadKey(vp, vaultKey).Return("Master of puppets is pulling your strings", nil).Times(1)
	return vaultCli
}

func vaultClientAndValidKey(ctrl *gomock.Controller) (*mocks.MockVaultClient, []byte) {
	vp := filepath.Join(testVaultPath, testFilename)
	vaultCli := mocks.NewMockVaultClient(ctrl)

	key := "one two three four"
	keyBytes := []byte(key)
	keyHexStr := hex.EncodeToString(keyBytes)

	vaultCli.EXPECT().ReadKey(vp, vaultKey).Return(keyHexStr, nil).Times(1)
	return vaultCli, keyBytes
}

func s3ClientGetWithPSKReturnsError(ctrl *gomock.Controller, key string, psk []byte) (*mocks.MockS3Client, error) {
	cli := mocks.NewMockS3Client(ctrl)
	cli.EXPECT().GetWithPSK(key, psk).Times(1).Return(nil, testErr)
	return cli, testErr
}

func s3ClientGetWithPSKReturnsReader(ctrl *gomock.Controller, key string, psk []byte, r S3ReadCloser) *mocks.MockS3Client {
	cli := mocks.NewMockS3Client(ctrl)
	cli.EXPECT().GetWithPSK(key, psk).Times(1).Return(r, nil)
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
