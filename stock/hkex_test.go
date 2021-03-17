package stock

import (
	"reflect"
	"testing"
)

func Test_getcompanyname(t *testing.T) {
	type args struct {
		c int
	}
	tests := []struct {
		name    string
		args    args
		want    Company
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getcompanyname(tt.args.c)
			if (err != nil) != tt.wantErr {
				t.Errorf("getcompanyname() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getcompanyname() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCompanyList(t *testing.T) {
	tests := []struct {
		name    string
		want    []int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCompanyList()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCompanyList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCompanyList() = %v, want %v", got, tt.want)
			}
		})
	}
}
