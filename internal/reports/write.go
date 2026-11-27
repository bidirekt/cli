package reports

import (
	"errors"
	"fmt"
	"io"

	"github.com/bidirekt/cli/internal/paint"
)

func WriteValidationFailed(errWriter io.Writer, err error, modules []Module) error {
	var validationFailed *ValidationFailedError
	if !errors.As(err, &validationFailed) {
		return err
	}

	if _, err := fmt.Fprint(errWriter, FormatValidationFailedReport(paint.For(errWriter), validationFailed.Message, validationFailed.Violations, modules)); err != nil {
		return err
	}

	return ErrSilent
}

func WriteVerdict(writer io.Writer, checkedSideLabel, participant, environment string, deployable bool, results map[string]CanIDeployResult, modules []Module) error {
	if err := refuseUnknownTrees(results, modules); err != nil {
		return err
	}

	if !deployable {
		report, err := FormatNotDeployableReport(paint.For(writer), checkedSideLabel, participant, environment, results, modules)
		if err != nil {
			return err
		}

		if _, err := fmt.Fprint(writer, report); err != nil {
			return err
		}

		return ErrSilent
	}

	if _, err := fmt.Fprint(writer, FormatDeployableReport(paint.For(writer), checkedSideLabel, environment, results)); err != nil {
		return err
	}

	return nil
}

func refuseUnknownTrees(results map[string]CanIDeployResult, modules []Module) error {
	for _, name := range sortedKeys(results) {
		for _, reportKey := range sortedKeys(results[name].Trees) {
			if _, err := protocolFor(reportKey, modules); err != nil {
				return err
			}
		}
	}

	return nil
}
