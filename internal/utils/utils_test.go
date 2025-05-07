// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"reflect"
	"testing"
)

func TestPruneEmpty(t *testing.T) {
	type args struct {
		list []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "empty",
			args: args{
				list: []string{},
			},
			want: []string{},
		},
		{
			name: "no op",
			args: args{
				list: []string{"foo", "bar"},
			},
			want: []string{"foo", "bar"},
		},
		{
			name: "front padded",
			args: args{
				list: []string{"", "", "foo", "bar"},
			},
			want: []string{"foo", "bar"},
		},
		{
			name: "back padded",
			args: args{
				list: []string{"foo", "bar", "", ""},
			},
			want: []string{"foo", "bar"},
		},
		{
			name: "middle padded",
			args: args{
				list: []string{"foo", "", "bar"},
			},
			want: []string{"foo", "bar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PruneEmpty(tt.args.list); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PruneEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
