// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// ControlPlaneMachineSetTemplateObjectMetaApplyConfiguration represents a declarative configuration of the ControlPlaneMachineSetTemplateObjectMeta type for use
// with apply.
type ControlPlaneMachineSetTemplateObjectMetaApplyConfiguration struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ControlPlaneMachineSetTemplateObjectMetaApplyConfiguration constructs a declarative configuration of the ControlPlaneMachineSetTemplateObjectMeta type for use with
// apply.
func ControlPlaneMachineSetTemplateObjectMeta() *ControlPlaneMachineSetTemplateObjectMetaApplyConfiguration {
	return &ControlPlaneMachineSetTemplateObjectMetaApplyConfiguration{}
}

// WithLabels puts the entries into the Labels field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Labels field,
// overwriting an existing map entries in Labels field with the same key.
func (b *ControlPlaneMachineSetTemplateObjectMetaApplyConfiguration) WithLabels(entries map[string]string) *ControlPlaneMachineSetTemplateObjectMetaApplyConfiguration {
	if b.Labels == nil && len(entries) > 0 {
		b.Labels = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.Labels[k] = v
	}
	return b
}

// WithAnnotations puts the entries into the Annotations field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Annotations field,
// overwriting an existing map entries in Annotations field with the same key.
func (b *ControlPlaneMachineSetTemplateObjectMetaApplyConfiguration) WithAnnotations(entries map[string]string) *ControlPlaneMachineSetTemplateObjectMetaApplyConfiguration {
	if b.Annotations == nil && len(entries) > 0 {
		b.Annotations = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.Annotations[k] = v
	}
	return b
}
