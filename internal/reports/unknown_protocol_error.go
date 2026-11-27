package reports

import "fmt"

func newUnknownProtocolError(reportKey string) error {
	return fmt.Errorf("the broker reported protocol %s, which this bidirekt doesn't know; update bidirekt", reportKey)
}
