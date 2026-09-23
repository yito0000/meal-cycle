package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// apiPrefix 配下は Wails IPC の代わりに使う E2E 専用の呼び出し口。
// 本番は @wailsio/runtime が /wails/runtime を叩くが、webview の外から
// 到達できないため、フロント側 shim と対になる HTTP 経路を用意している。
const apiPrefix = "/api/"

// serviceRegistry はサービス名から Wails 登録済みインスタンスを引く。
// main.go の application.Options.Services と同じ 4 つを保つこと。
type serviceRegistry map[string]any

// dispatchHandler は POST /api/{Service}/{Method} を reflect でサービスに転送する。
func dispatchHandler(reg serviceRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "POST only")
			return
		}

		serviceName, methodName, ok := splitAPIPath(r.URL.Path)
		if !ok {
			writeError(w, http.StatusNotFound, "ReferenceError", "malformed api path")
			return
		}

		svc, ok := reg[serviceName]
		if !ok {
			writeError(w, http.StatusNotFound, "ReferenceError", "unknown service "+serviceName)
			return
		}

		method := reflect.ValueOf(svc).MethodByName(methodName)
		if !method.IsValid() {
			writeError(w, http.StatusNotFound, "ReferenceError", "unknown method "+methodName)
			return
		}

		var rawArgs []json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&rawArgs); err != nil {
			writeError(w, http.StatusBadRequest, "TypeError", "malformed body: "+err.Error())
			return
		}

		in, err := buildArgs(r.Context(), method.Type(), rawArgs)
		if err != nil {
			writeError(w, http.StatusBadRequest, "TypeError", err.Error())
			return
		}

		writeResult(w, method.Call(in))
	})
}

// splitAPIPath は "/api/RecipeService/List" を ("RecipeService", "List") に割る。
func splitAPIPath(path string) (service, method string, ok bool) {
	rest := strings.TrimPrefix(path, apiPrefix)
	if rest == path {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

var ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()

// buildArgs は JSON 引数をメソッドのシグネチャに合わせて復元する。
// 第 1 引数が context.Context のメソッド (大半) と、取らないメソッド
// (HealthService.Ping) の両方を受ける。
func buildArgs(ctx context.Context, mt reflect.Type, rawArgs []json.RawMessage) ([]reflect.Value, error) {
	in := make([]reflect.Value, 0, mt.NumIn())
	next := 0
	if mt.NumIn() > 0 && mt.In(0) == ctxType {
		in = append(in, reflect.ValueOf(ctx))
		next = 1
	}

	want := mt.NumIn() - next
	if len(rawArgs) != want {
		return nil, fmt.Errorf("want %d args, got %d", want, len(rawArgs))
	}

	for i := 0; i < want; i++ {
		p := reflect.New(mt.In(next + i))
		if err := json.Unmarshal(rawArgs[i], p.Interface()); err != nil {
			return nil, fmt.Errorf("arg %d: %w", i, err)
		}
		in = append(in, p.Elem())
	}
	return in, nil
}

var errType = reflect.TypeOf((*error)(nil)).Elem()

// writeResult は戻り値末尾の error を HTTP ステータスに、残りを JSON body に写す。
func writeResult(w http.ResponseWriter, out []reflect.Value) {
	if n := len(out); n > 0 && out[n-1].Type() == errType {
		if !out[n-1].IsNil() {
			err := out[n-1].Interface().(error)
			writeError(w, http.StatusInternalServerError, errorKind(err), err.Error())
			return
		}
		out = out[:n-1]
	}

	var body any
	if len(out) > 0 {
		body = out[0].Interface()
	}
	writeJSON(w, http.StatusOK, body)
}

// errorKind は shim 側でエラーを識別するための型名。@wailsio/runtime の
// 既定トランスポートが kind で分岐するのに合わせている。
func errorKind(err error) string {
	t := reflect.TypeOf(err)
	if t == nil {
		return "Error"
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Name() == "" {
		return "Error"
	}
	return t.Name()
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func writeError(w http.ResponseWriter, status int, kind, message string) {
	writeJSON(w, status, map[string]string{"kind": kind, "message": message})
}
