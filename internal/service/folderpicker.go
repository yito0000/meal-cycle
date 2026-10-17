package service

import "github.com/wailsapp/wails/v3/pkg/application"

// FolderPicker は書き出し先のフォルダを利用者に選ばせる。E2E とテストで差し替えるため interface にする (add-recipe-export D1)。
type FolderPicker interface {
	// 閉じられたら "" と nil を返す。
	PickFolder(title string) (string, error)
}

type wailsFolderPicker struct{}

func NewWailsFolderPicker() FolderPicker { return wailsFolderPicker{} }

func (wailsFolderPicker) PickFolder(title string) (string, error) {
	return application.Get().Dialog.OpenFile().
		SetTitle(title).
		CanChooseFiles(false).
		CanChooseDirectories(true).
		CanCreateDirectories(true).
		PromptForSingleSelection()
}
