// Command apigen derives the public OpenAPI schemas from the Go wire types.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Kirill555dg/mini-games/internal/availability"
	"github.com/Kirill555dg/mini-games/internal/battleship"
	"github.com/Kirill555dg/mini-games/internal/engine"
	"github.com/Kirill555dg/mini-games/internal/history"
	"github.com/Kirill555dg/mini-games/internal/httpapi"
	"github.com/Kirill555dg/mini-games/internal/rooms"
)

type schema map[string]any

var components = map[string]schema{}

func describe(t reflect.Type) schema {
	if t == reflect.TypeOf(time.Time{}) {
		return schema{"type": "string", "format": "date-time"}
	}
	if t.Kind() == reflect.Pointer {
		return describe(t.Elem())
	}
	switch t.Kind() {
	case reflect.Bool:
		return schema{"type": "boolean"}
	case reflect.String:
		return schema{"type": "string"}
	case reflect.Int, reflect.Int64, reflect.Uint64, reflect.Int32, reflect.Uint32:
		return schema{"type": "integer"}
	case reflect.Slice, reflect.Array:
		return schema{"type": "array", "items": describe(t.Elem())}
	case reflect.Map:
		return schema{"type": "object", "additionalProperties": describe(t.Elem())}
	case reflect.Struct:
		name := filepath.Base(t.PkgPath()) + "_" + t.Name()
		ref := schema{"$ref": "#/components/schemas/" + name}
		if _, ok := components[name]; ok {
			return ref
		}
		components[name] = schema{}
		properties := map[string]schema{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")
			if f.PkgPath != "" || tag[0] == "-" {
				continue
			}
			key := tag[0]
			if key == "" {
				key = f.Name
			}
			properties[key] = describe(f.Type)
			if !strings.Contains(f.Tag.Get("json"), "omitempty") {
				required = append(required, key)
			}
			if name == "rooms_Command" && key == "action" {
				values := []string{}
				for _, value := range rooms.Actions() {
					values = append(values, value)
				}
				sort.Strings(values)
				properties[key]["enum"] = values
			}
			if name == "availability_Day" && key == "state" {
				properties[key]["enum"] = availability.States()
			}
		}
		components[name] = schema{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
		return ref
	}
	panic("unsupported API type: " + t.String())
}
func main() {
	paths := map[string]any{}
	add := func(path, method, id string, result any, body any, room bool) {
		op := schema{"operationId": id, "responses": map[string]any{"200": schema{"description": "Success", "content": schema{"application/json": schema{"schema": describe(reflect.TypeOf(result))}}}, "default": schema{"description": "Request rejected", "content": schema{"application/json": schema{"schema": describe(reflect.TypeOf(httpapi.ErrorResponse{}))}}}}}
		if body != nil {
			op["requestBody"] = schema{"required": true, "content": schema{"application/json": schema{"schema": describe(reflect.TypeOf(body))}}}
		}
		if room {
			op["parameters"] = []any{schema{"name": "room", "in": "query", "required": true, "schema": schema{"type": "string", "pattern": fmt.Sprintf("^[%s]{%d}$", rooms.CodeAlphabet, rooms.CodeLength)}}}
		}
		paths[path] = map[string]any{method: op}
	}
	add("/game-api/lobby", "get", "getLobby", rooms.Lobby{}, nil, false)
	add("/game-api/state", "get", "getRoom", rooms.View{}, nil, true)
	add("/game-api/action", "post", "sendAction", rooms.View{}, rooms.Command{}, false)
	add("/game-api/history", "get", "getHistory", []history.Record{}, nil, false)
	add("/game-api/status", "get", "getStatus", httpapi.Status{}, nil, false)
	add("/game-api/visit", "post", "trackVisit", httpapi.VisitResponse{}, nil, false)
	paths["/game-api/events"] = map[string]any{"get": schema{"operationId": "events", "parameters": []any{schema{"name": "room", "in": "query", "required": false, "schema": schema{"type": "string", "pattern": fmt.Sprintf("^[%s]{%d}$", rooms.CodeAlphabet, rooms.CodeLength)}}}, "responses": schema{"200": schema{"description": "SSE state events: Lobby or viewer-specific View; null means room closed", "content": schema{"text/event-stream": schema{"schema": schema{"type": "string"}}}}}}}
	_ = describe(reflect.TypeOf(engine.Spec{}))
	document := schema{"x-naval-cells": battleship.Cells(), "x-room-actions": rooms.Actions(), "x-room-code": schema{"length": rooms.CodeLength, "alphabet": rooms.CodeAlphabet, "maxNameRunes": rooms.MaxNameRunes}, "x-game-values": schema{"draw": engine.Draw, "noWinner": engine.NoWinner, "playing": engine.Playing, "setup": engine.Setup, "finished": engine.Finished}, "openapi": "3.1.0", "info": schema{"title": "Mini-games API", "version": "1.0.0", "license": schema{"name": "MIT", "url": "https://opensource.org/license/mit/"}}, "servers": []any{schema{"url": "/"}}, "paths": paths, "components": schema{"schemas": components}}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		panic(err)
	}
	if err = os.MkdirAll("api", 0755); err != nil {
		panic(err)
	}
	if err = os.WriteFile("api/openapi.json", append(data, '\n'), 0644); err != nil {
		panic(err)
	}
}
