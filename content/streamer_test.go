package content

import (
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ONSdigital/dp-download-service/content/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testVaultPath = "wibble"
	testFilename  = "1234567890.csv"
	testErr       = errors.New("bork")
)

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

		vaultCli := vaultClientErrorOnReadKey(ctrl)

		s := New(vaultCli, testVaultPath)
		psk, err := s.getVaultKeyForFile(testFilename)

		So(psk, ShouldBeNil)
		So(err, ShouldNotBeNil)
		So(err, ShouldEqual, testErr)
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

func vaultClientErrorOnReadKey(ctrl *gomock.Controller) *mocks.MockVaultClient {
	vp := filepath.Join(testVaultPath, testFilename)
	vaultCli := mocks.NewMockVaultClient(ctrl)
	vaultCli.EXPECT().ReadKey(vp, vaultKey).Return("", testErr).Times(1)
	return vaultCli
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
