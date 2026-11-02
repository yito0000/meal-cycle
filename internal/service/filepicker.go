package service

import "github.com/wailsapp/wails/v3/pkg/application"

// FilePicker は書き出し先と復元に使うファイルを利用者に選ばせる。E2E とテストで差し替えるため interface にする
// (add-settings-database-file design.md D5)。どちらも閉じられたら "" と nil を返す。
type FilePicker interface {
	SaveFile(title, defaultName string) (string, error)
	OpenFile(title string) (string, error)
}

type wailsFilePicker struct{}

func NewWailsFilePicker() FilePicker { return wailsFilePicker{} }

func (wailsFilePicker) SaveFile(title, defaultName string) (string, error) {
	return application.Get().Dialog.SaveFile().
		SetMessage(title).
		SetFilename(defaultName).
		AddFilter("meal-cycle", "*.db").
		CanCreateDirectories(true).
		PromptForSingleSelection()
}

func (wailsFilePicker) OpenFile(title string) (string, error) {
	return application.Get().Dialog.OpenFile().
		SetTitle(title).
		AddFilter("meal-cycle", "*.db").
		CanChooseFiles(true).
		CanChooseDirectories(false).
		PromptForSingleSelection()
}
