package reports

import (
	"errors"
	"fmt"
	"io"

	"github.com/bidirekt/cli/internal/paint"
)

func WriteValidationFailed(errWriter io.Writer, err error) error {
	var validationFailed *ValidationFailedError
	if !errors.As(err, &validationFailed) {
		return err
	}

	if _, err := fmt.Fprint(errWriter, FormatValidationFailedReport(paint.For(errWriter), validationFailed.Message, validationFailed.Violations)); err != nil {
		return err
	}

	return ErrSilent
}

func WriteVerdict(writer io.Writer, checkedSideLabel, participant, environment string, deployable bool, results map[string]CanIDeployResult) error {
	if !deployable {
		if _, err := fmt.Fprint(writer, FormatNotDeployableReport(paint.For(writer), checkedSideLabel, participant, environment, results)); err != nil {
			return err
		}

		return ErrSilent
	}

	if _, err := fmt.Fprintln(writer, FormatDeployableLine(paint.For(writer), checkedSideLabel, environment)); err != nil {
		return err
	}

	return nil
}
