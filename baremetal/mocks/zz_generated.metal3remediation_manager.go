// /*
// Copyright 2011-2016 Canonical Ltd.
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
// */
//

// Code generated by MockGen. DO NOT EDIT.
// Source: ./baremetal/metal3remediation_manager.go

// Package baremetal_mocks is a generated GoMock package.
package baremetal_mocks

import (
	context "context"
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	v1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	v1beta1 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1beta10 "sigs.k8s.io/cluster-api/api/v1beta1"
	patch "sigs.k8s.io/cluster-api/util/patch"
)

// MockRemediationManagerInterface is a mock of RemediationManagerInterface interface.
type MockRemediationManagerInterface struct {
	ctrl     *gomock.Controller
	recorder *MockRemediationManagerInterfaceMockRecorder
}

// MockRemediationManagerInterfaceMockRecorder is the mock recorder for MockRemediationManagerInterface.
type MockRemediationManagerInterfaceMockRecorder struct {
	mock *MockRemediationManagerInterface
}

// NewMockRemediationManagerInterface creates a new mock instance.
func NewMockRemediationManagerInterface(ctrl *gomock.Controller) *MockRemediationManagerInterface {
	mock := &MockRemediationManagerInterface{ctrl: ctrl}
	mock.recorder = &MockRemediationManagerInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRemediationManagerInterface) EXPECT() *MockRemediationManagerInterfaceMockRecorder {
	return m.recorder
}

// GetCapiMachine mocks base method.
func (m *MockRemediationManagerInterface) GetCapiMachine(ctx context.Context) (*v1beta10.Machine, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCapiMachine", ctx)
	ret0, _ := ret[0].(*v1beta10.Machine)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCapiMachine indicates an expected call of GetCapiMachine.
func (mr *MockRemediationManagerInterfaceMockRecorder) GetCapiMachine(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCapiMachine", reflect.TypeOf((*MockRemediationManagerInterface)(nil).GetCapiMachine), ctx)
}

// GetLastRemediatedTime mocks base method.
func (m *MockRemediationManagerInterface) GetLastRemediatedTime() *v1.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastRemediatedTime")
	ret0, _ := ret[0].(*v1.Time)
	return ret0
}

// GetLastRemediatedTime indicates an expected call of GetLastRemediatedTime.
func (mr *MockRemediationManagerInterfaceMockRecorder) GetLastRemediatedTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastRemediatedTime", reflect.TypeOf((*MockRemediationManagerInterface)(nil).GetLastRemediatedTime))
}

// GetRemediationPhase mocks base method.
func (m *MockRemediationManagerInterface) GetRemediationPhase() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRemediationPhase")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetRemediationPhase indicates an expected call of GetRemediationPhase.
func (mr *MockRemediationManagerInterfaceMockRecorder) GetRemediationPhase() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRemediationPhase", reflect.TypeOf((*MockRemediationManagerInterface)(nil).GetRemediationPhase))
}

// GetRemediationType mocks base method.
func (m *MockRemediationManagerInterface) GetRemediationType() v1beta1.RemediationType {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRemediationType")
	ret0, _ := ret[0].(v1beta1.RemediationType)
	return ret0
}

// GetRemediationType indicates an expected call of GetRemediationType.
func (mr *MockRemediationManagerInterfaceMockRecorder) GetRemediationType() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRemediationType", reflect.TypeOf((*MockRemediationManagerInterface)(nil).GetRemediationType))
}

// GetTimeout mocks base method.
func (m *MockRemediationManagerInterface) GetTimeout() *v1.Duration {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTimeout")
	ret0, _ := ret[0].(*v1.Duration)
	return ret0
}

// GetTimeout indicates an expected call of GetTimeout.
func (mr *MockRemediationManagerInterfaceMockRecorder) GetTimeout() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTimeout", reflect.TypeOf((*MockRemediationManagerInterface)(nil).GetTimeout))
}

// GetUnhealthyHost mocks base method.
func (m *MockRemediationManagerInterface) GetUnhealthyHost(ctx context.Context) (*v1alpha1.BareMetalHost, *patch.Helper, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUnhealthyHost", ctx)
	ret0, _ := ret[0].(*v1alpha1.BareMetalHost)
	ret1, _ := ret[1].(*patch.Helper)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetUnhealthyHost indicates an expected call of GetUnhealthyHost.
func (mr *MockRemediationManagerInterfaceMockRecorder) GetUnhealthyHost(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUnhealthyHost", reflect.TypeOf((*MockRemediationManagerInterface)(nil).GetUnhealthyHost), ctx)
}

// HasReachRetryLimit mocks base method.
func (m *MockRemediationManagerInterface) HasReachRetryLimit() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasReachRetryLimit")
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasReachRetryLimit indicates an expected call of HasReachRetryLimit.
func (mr *MockRemediationManagerInterfaceMockRecorder) HasReachRetryLimit() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasReachRetryLimit", reflect.TypeOf((*MockRemediationManagerInterface)(nil).HasReachRetryLimit))
}

// IncreaseRetryCount mocks base method.
func (m *MockRemediationManagerInterface) IncreaseRetryCount() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "IncreaseRetryCount")
}

// IncreaseRetryCount indicates an expected call of IncreaseRetryCount.
func (mr *MockRemediationManagerInterfaceMockRecorder) IncreaseRetryCount() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IncreaseRetryCount", reflect.TypeOf((*MockRemediationManagerInterface)(nil).IncreaseRetryCount))
}

// OnlineStatus mocks base method.
func (m *MockRemediationManagerInterface) OnlineStatus(host *v1alpha1.BareMetalHost) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OnlineStatus", host)
	ret0, _ := ret[0].(bool)
	return ret0
}

// OnlineStatus indicates an expected call of OnlineStatus.
func (mr *MockRemediationManagerInterfaceMockRecorder) OnlineStatus(host interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnlineStatus", reflect.TypeOf((*MockRemediationManagerInterface)(nil).OnlineStatus), host)
}

// RetryLimitIsSet mocks base method.
func (m *MockRemediationManagerInterface) RetryLimitIsSet() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RetryLimitIsSet")
	ret0, _ := ret[0].(bool)
	return ret0
}

// RetryLimitIsSet indicates an expected call of RetryLimitIsSet.
func (mr *MockRemediationManagerInterfaceMockRecorder) RetryLimitIsSet() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RetryLimitIsSet", reflect.TypeOf((*MockRemediationManagerInterface)(nil).RetryLimitIsSet))
}

// SetFinalizer mocks base method.
func (m *MockRemediationManagerInterface) SetFinalizer() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetFinalizer")
}

// SetFinalizer indicates an expected call of SetFinalizer.
func (mr *MockRemediationManagerInterfaceMockRecorder) SetFinalizer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetFinalizer", reflect.TypeOf((*MockRemediationManagerInterface)(nil).SetFinalizer))
}

// SetLastRemediationTime mocks base method.
func (m *MockRemediationManagerInterface) SetLastRemediationTime(remediationTime *v1.Time) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetLastRemediationTime", remediationTime)
}

// SetLastRemediationTime indicates an expected call of SetLastRemediationTime.
func (mr *MockRemediationManagerInterfaceMockRecorder) SetLastRemediationTime(remediationTime interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetLastRemediationTime", reflect.TypeOf((*MockRemediationManagerInterface)(nil).SetLastRemediationTime), remediationTime)
}

// SetOwnerRemediatedConditionNew mocks base method.
func (m *MockRemediationManagerInterface) SetOwnerRemediatedConditionNew(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetOwnerRemediatedConditionNew", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetOwnerRemediatedConditionNew indicates an expected call of SetOwnerRemediatedConditionNew.
func (mr *MockRemediationManagerInterfaceMockRecorder) SetOwnerRemediatedConditionNew(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetOwnerRemediatedConditionNew", reflect.TypeOf((*MockRemediationManagerInterface)(nil).SetOwnerRemediatedConditionNew), ctx)
}

// SetRebootAnnotation mocks base method.
func (m *MockRemediationManagerInterface) SetRebootAnnotation(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetRebootAnnotation", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetRebootAnnotation indicates an expected call of SetRebootAnnotation.
func (mr *MockRemediationManagerInterfaceMockRecorder) SetRebootAnnotation(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetRebootAnnotation", reflect.TypeOf((*MockRemediationManagerInterface)(nil).SetRebootAnnotation), ctx)
}

// SetRemediationPhase mocks base method.
func (m *MockRemediationManagerInterface) SetRemediationPhase(phase string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetRemediationPhase", phase)
}

// SetRemediationPhase indicates an expected call of SetRemediationPhase.
func (mr *MockRemediationManagerInterfaceMockRecorder) SetRemediationPhase(phase interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetRemediationPhase", reflect.TypeOf((*MockRemediationManagerInterface)(nil).SetRemediationPhase), phase)
}

// SetUnhealthyAnnotation mocks base method.
func (m *MockRemediationManagerInterface) SetUnhealthyAnnotation(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetUnhealthyAnnotation", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetUnhealthyAnnotation indicates an expected call of SetUnhealthyAnnotation.
func (mr *MockRemediationManagerInterfaceMockRecorder) SetUnhealthyAnnotation(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetUnhealthyAnnotation", reflect.TypeOf((*MockRemediationManagerInterface)(nil).SetUnhealthyAnnotation), ctx)
}

// TimeToRemediate mocks base method.
func (m *MockRemediationManagerInterface) TimeToRemediate(timeout time.Duration) (bool, time.Duration) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TimeToRemediate", timeout)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(time.Duration)
	return ret0, ret1
}

// TimeToRemediate indicates an expected call of TimeToRemediate.
func (mr *MockRemediationManagerInterfaceMockRecorder) TimeToRemediate(timeout interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TimeToRemediate", reflect.TypeOf((*MockRemediationManagerInterface)(nil).TimeToRemediate), timeout)
}

// UnsetFinalizer mocks base method.
func (m *MockRemediationManagerInterface) UnsetFinalizer() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UnsetFinalizer")
}

// UnsetFinalizer indicates an expected call of UnsetFinalizer.
func (mr *MockRemediationManagerInterfaceMockRecorder) UnsetFinalizer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnsetFinalizer", reflect.TypeOf((*MockRemediationManagerInterface)(nil).UnsetFinalizer))
}
