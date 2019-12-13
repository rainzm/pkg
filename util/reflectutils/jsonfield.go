// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reflectutils

import (
	"reflect"
	"strings"
	"sync"

	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"
)

type SStructFieldInfo struct {
	Ignore      bool
	OmitEmpty   bool
	OmitFalse   bool
	OmitZero    bool
	Name        string
	FieldName   string
	ForceString bool
	Tags        map[string]string
}

func ParseStructFieldJsonInfo(sf reflect.StructField) SStructFieldInfo {
	info := SStructFieldInfo{}
	info.FieldName = sf.Name
	info.OmitEmpty = true
	info.OmitZero = false
	info.OmitFalse = false

	info.Tags = utils.TagMap(sf.Tag)
	if val, ok := info.Tags["json"]; ok {
		keys := strings.Split(val, ",")
		if len(keys) > 0 {
			if keys[0] == "-" {
				if len(keys) > 1 {
					info.Name = keys[0]
				} else {
					info.Ignore = true
				}
			} else {
				info.Name = keys[0]
			}
		}
		if len(keys) > 1 {
			for _, k := range keys[1:] {
				switch strings.ToLower(k) {
				case "omitempty":
					info.OmitEmpty = true
				case "allowempty":
					info.OmitEmpty = false
				case "omitzero":
					info.OmitZero = true
				case "allowzero":
					info.OmitZero = false
				case "omitfalse":
					info.OmitFalse = true
				case "allowfalse":
					info.OmitFalse = false
				case "string":
					info.ForceString = true
				}
			}
		}
	}
	if val, ok := info.Tags["name"]; ok {
		info.Name = val
	}
	return info
}

func (info *SStructFieldInfo) MarshalName() string {
	if len(info.Name) > 0 {
		return info.Name
	}
	info.Name = utils.CamelSplit(info.FieldName, "_")
	return info.Name
}

type SStructFieldValue struct {
	Info  SStructFieldInfo
	Value reflect.Value
}

type SStructFieldValueSet []SStructFieldValue

func FetchStructFieldValueSet(dataValue reflect.Value) SStructFieldValueSet {
	return fetchStructFieldValueSet(dataValue, false)
}

func FetchStructFieldValueSetForWrite(dataValue reflect.Value) SStructFieldValueSet {
	return fetchStructFieldValueSet(dataValue, true)
}

func fetchStructFieldValueSet(dataValue reflect.Value, allocatePtr bool) SStructFieldValueSet {
	fields := SStructFieldValueSet{}
	dataType := dataValue.Type()
	for i := 0; i < dataType.NumField(); i += 1 {
		sf := dataType.Field(i)

		// ignore unexported field altogether
		if !gotypes.IsFieldExportable(sf.Name) {
			continue
		}
		fv := dataValue.Field(i)
		if !fv.IsValid() {
			continue
		}
		if sf.Anonymous {
			// T, *T
			switch fv.Kind() {
			case reflect.Ptr, reflect.Interface:
				if !fv.IsValid() {
					continue
				}
				if fv.IsNil() {
					if fv.Kind() == reflect.Ptr && allocatePtr {
						fv.Set(reflect.New(fv.Type().Elem()))
					} else {
						continue
					}
				}
				fv = fv.Elem()
			}
			// note that we regard anonymous interface field the
			// same as with anonymous struct field.  This is
			// different from how encoding/json handles struct
			// field of interface type.
			if fv.Kind() == reflect.Struct && sf.Type != gotypes.TimeType {
				subfields := fetchStructFieldValueSet(fv, allocatePtr)
				fields = append(fields, subfields...)
				continue
			}
		}
		jsonInfo := ParseStructFieldJsonInfo(sf)
		fields = append(fields, SStructFieldValue{
			Info:  jsonInfo,
			Value: fv,
		})
	}
	return fields
}

func (set SStructFieldValueSet) GetStructFieldIndex(name string) int {
	for i := 0; i < len(set); i += 1 {
		jsonInfo := set[i].Info
		if jsonInfo.MarshalName() == name {
			return i
		}
		if utils.CamelSplit(jsonInfo.FieldName, "_") == utils.CamelSplit(name, "_") {
			return i
		}
		if jsonInfo.FieldName == name {
			return i
		}
		if jsonInfo.FieldName == utils.Capitalize(name) {
			return i
		}
	}
	return -1
}

func (set SStructFieldValueSet) GetValue(name string) (reflect.Value, bool) {
	idx := set.GetStructFieldIndex(name)
	if idx < 0 {
		return reflect.Value{}, false
	}
	return set[idx].Value, true
}

func (set SStructFieldValueSet) GetInterface(name string) (interface{}, bool) {
	idx := set.GetStructFieldIndex(name)
	if idx < 0 {
		return nil, false
	}
	if set[idx].Value.CanInterface() {
		return set[idx].Value.Interface(), true
	}
	return nil, false
}

func (set SStructFieldValueSetV2) GetValue(name string) (reflect.Value, bool) {
	idx := set.getStructFieldIndex(name)
	if idx < 0 {
		return reflect.Value{}, false
	}
	return set.Values[idx].Value, true
}

func (set SStructFieldValueSetV2) GetInterface(name string) (interface{}, bool) {
	idx := set.getStructFieldIndex(name)
	if idx < 0 {
		return nil, false
	}
	if set.Values[idx].Value.CanInterface() {
		return set.Values[idx].Value.Interface(), true
	}
	return nil, false
}

func (set SStructFieldValueSetV2) getStructFieldIndex(name string) int {
	index := -1
	for i, jsonInfo := range set.Infos {
		if jsonInfo.MarshalName() == name {
			index = i
			break
		}
		if utils.CamelSplit(jsonInfo.FieldName, "_") == utils.CamelSplit(name, "_") {
			index = i
			break
		}
		if jsonInfo.FieldName == name {
			index = i
			break
		}
		if jsonInfo.FieldName == utils.Capitalize(name) {
			index = i
			break
		}
	}
	if index < 0 {
		return index
	}
	if index >= len(set.Values) {
		index = len(set.Values) - 1
	}
	// Reverse traversal from index will be faster
	for i := index; i >= 0; i-- {
		if set.Values[i].Index == index {
			return i
		}
	}
	// no arrival
	return index
}

type SStructFieldValueSetV2 struct {
	Infos []SStructFieldInfo
	Values []SStructFieldValueV2
}

type SStructFieldValueV2 struct {
	Value reflect.Value
	Index int
}

func FetchStructFieldValueSetV2(dataValue reflect.Value) SStructFieldValueSetV2 {
	infos := cachefetchStructFieldInfos(dataValue)
	values, _ := fetchStructFieldValueV2s(dataValue, false, 0)
	return SStructFieldValueSetV2{infos, values}
}

func FetchStructFieldValueSetForWriteV2(dataValue reflect.Value) SStructFieldValueSetV2 {
	infos := cachefetchStructFieldInfos(dataValue)
	values, _ := fetchStructFieldValueV2s(dataValue, true, 0)
	return SStructFieldValueSetV2{infos, values}
}

var structFieldInfoCache sync.Map

func cachefetchStructFieldInfos(dataValue reflect.Value) []SStructFieldInfo {
	dataType := dataValue.Type()
	if r, ok := structFieldInfoCache.Load(dataType); ok {
		return r.([]SStructFieldInfo)
	}
	f, _ := structFieldInfoCache.LoadOrStore(dataType, fetchStructFieldInfos(dataValue))
	return f.([]SStructFieldInfo)
}

func fetchStructFieldInfos(dataValue reflect.Value) []SStructFieldInfo {
	dataType := dataValue.Type()
	ret := make([]SStructFieldInfo, 0, dataType.NumField())
	for i := 0; i < dataType.NumField(); i += 1 {
		sf := dataType.Field(i)
		if !gotypes.IsFieldExportable(sf.Name) {
			continue
		}
		fv := dataValue.Field(i)
		if sf.Anonymous {
			if !fv.IsValid() {
				continue
			}
			if fv.Kind() == reflect.Ptr {
				if fv.IsNil() {
					fv.Set(reflect.New(fv.Type().Elem()))
				}
				fv = fv.Elem()
			}
			if fv.Kind() == reflect.Interface {
				fv = fv.Elem()
			}
			if fv.Kind() == reflect.Struct && sf.Type != gotypes.TimeType{
			 	subInfo := fetchStructFieldInfos(fv)
			 	ret = append(ret, subInfo...)
			 	continue
		 }
		}
		jsonInfo := ParseStructFieldJsonInfo(sf)
		ret = append(ret, jsonInfo)
	}
	return ret[:len(ret):len(ret)]
}

func fetchStructFieldValueV2s(dataValue reflect.Value, allocatePtr bool, index int) ([]SStructFieldValueV2, int) {
	dataType := dataValue.Type()
	ret := make([]SStructFieldValueV2, 0, dataType.NumField())
	for i := 0; i < dataType.NumField(); i += 1 {
		sf := dataType.Field(i)

		if !gotypes.IsFieldExportable(sf.Name) {
			continue
		}
		fv := dataValue.Field(i)
		if sf.Anonymous {
			switch fv.Kind() {
			case reflect.Ptr, reflect.Interface:
				if !fv.IsValid() {
					continue
				}
				if fv.IsNil() {
					if fv.Kind() == reflect.Ptr && allocatePtr {
						fv.Set(reflect.New(fv.Type().Elem()))
					} else {
						index += 1
						continue
					}
				}
				fv = fv.Elem()
			}
			if fv.Kind() == reflect.Struct && sf.Type != gotypes.TimeType {
				var subValues []SStructFieldValueV2
				subValues, index = fetchStructFieldValueV2s(fv, allocatePtr, index)
				ret = append(ret, subValues...)
				continue
			}
		}
		ret = append(ret, SStructFieldValueV2{fv, index})
		index += 1
	}
	return ret, index
}
