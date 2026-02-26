/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhook

import (
	"context"
	"fmt"
	"time"

	infrav1 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	crwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	defaultDuration = 600 * time.Second
	minDuration     = 100 * time.Second
	minRetryLimit   = 1
)

var (
	defaultTimeout = metav1.Duration{Duration: defaultDuration}
	minTimeout     = metav1.Duration{Duration: minDuration}
)

// Metal3Remediation implements validation and defaulting webhooks for Metal3Remediation.
type Metal3Remediation struct{}

var _ crwebhook.CustomDefaulter = &Metal3Remediation{}
var _ crwebhook.CustomValidator = &Metal3Remediation{}

func (w *Metal3Remediation) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&infrav1.Metal3Remediation{}).
		WithDefaulter(w, admission.DefaulterRemoveUnknownOrOmitableFields).
		WithValidator(w).
		Complete()
}

func (w *Metal3Remediation) Default(_ context.Context, _ runtime.Object) error {
	return nil
}

func (w *Metal3Remediation) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	c, ok := obj.(*infrav1.Metal3Remediation)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a Metal3Remediation but got a %T", obj))
	}
	return nil, validateRemediation(c)
}

func (w *Metal3Remediation) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	c, ok := newObj.(*infrav1.Metal3Remediation)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a Metal3Remediation but got a %T", newObj))
	}
	return nil, validateRemediation(c)
}

func (w *Metal3Remediation) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateRemediation(r *infrav1.Metal3Remediation) error {
	var allErrs field.ErrorList
	if r.Spec.Strategy.Timeout != nil && r.Spec.Strategy.Timeout.Seconds() < minTimeout.Seconds() {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "strategy", "timeout"),
			r.Spec.Strategy.Timeout,
			fmt.Sprintf("min duration is %s", minTimeout.Duration),
		))
	}

	if r.Spec.Strategy.Type != infrav1.RebootRemediationStrategy {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "strategy", "type"),
			r.Spec.Strategy.Type,
			"only supported remediation strategy is reboot",
		))
	}

	if r.Spec.Strategy.RetryLimit < minRetryLimit {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "strategy", "retryLimit"),
			r.Spec.Strategy.RetryLimit,
			fmt.Sprintf("minimum retry limit is %d", minRetryLimit),
		))
	}

	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(infrav1.GroupVersion.WithKind("Metal3Remediation").GroupKind(), r.Name, allErrs)
}
