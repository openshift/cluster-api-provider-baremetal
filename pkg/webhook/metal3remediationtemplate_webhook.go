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

	infrav1 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	crwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Metal3RemediationTemplate implements validation and defaulting webhooks for Metal3RemediationTemplate.
type Metal3RemediationTemplate struct{}

var _ crwebhook.CustomDefaulter = &Metal3RemediationTemplate{}
var _ crwebhook.CustomValidator = &Metal3RemediationTemplate{}

func (w *Metal3RemediationTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&infrav1.Metal3RemediationTemplate{}).
		WithDefaulter(w, admission.DefaulterRemoveUnknownOrOmitableFields).
		WithValidator(w).
		Complete()
}

func (w *Metal3RemediationTemplate) Default(_ context.Context, obj runtime.Object) error {
	m3rt, ok := obj.(*infrav1.Metal3RemediationTemplate)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Metal3RemediationTemplate but got a %T", obj))
	}

	if m3rt.Spec.Template.Spec.Strategy.Type == "" {
		m3rt.Spec.Template.Spec.Strategy.Type = infrav1.RebootRemediationStrategy
	}

	if m3rt.Spec.Template.Spec.Strategy.Timeout == nil {
		m3rt.Spec.Template.Spec.Strategy.Timeout = &defaultTimeout
	}

	if m3rt.Spec.Template.Spec.Strategy.RetryLimit < minRetryLimit {
		m3rt.Spec.Template.Spec.Strategy.RetryLimit = minRetryLimit
	}

	return nil
}

func (w *Metal3RemediationTemplate) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	m3rt, ok := obj.(*infrav1.Metal3RemediationTemplate)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a Metal3RemediationTemplate but got a %T", obj))
	}
	return nil, validateRemediationTemplate(m3rt)
}

func (w *Metal3RemediationTemplate) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	m3rt, ok := newObj.(*infrav1.Metal3RemediationTemplate)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a Metal3RemediationTemplate but got a %T", newObj))
	}
	return nil, validateRemediationTemplate(m3rt)
}

func (w *Metal3RemediationTemplate) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateRemediationTemplate(m3rt *infrav1.Metal3RemediationTemplate) error {
	var allErrs field.ErrorList

	if m3rt.Spec.Template.Spec.Strategy.Timeout != nil && m3rt.Spec.Template.Spec.Strategy.Timeout.Seconds() < minTimeout.Seconds() {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "template", "spec", "strategy", "timeout"),
			m3rt.Spec.Template.Spec.Strategy.Timeout,
			fmt.Sprintf("min duration is %s", minTimeout.Duration),
		))
	}

	if m3rt.Spec.Template.Spec.Strategy.Type != infrav1.RebootRemediationStrategy {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "template", "spec", "strategy", "type"),
			m3rt.Spec.Template.Spec.Strategy.Type,
			"only supported remediation strategy is reboot",
		))
	}

	if m3rt.Spec.Template.Spec.Strategy.RetryLimit < minRetryLimit {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "template", "spec", "strategy", "retryLimit"),
			m3rt.Spec.Template.Spec.Strategy.RetryLimit,
			fmt.Sprintf("minimum retry limit is %d", minRetryLimit),
		))
	}

	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(infrav1.GroupVersion.WithKind("Metal3RemediationTemplate").GroupKind(), m3rt.Name, allErrs)
}
