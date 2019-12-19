package testsuite

import (
	"fmt"
	"github.com/avast/retry-go"
	acApi "github.com/kyma-project/kyma/components/application-operator/pkg/apis/applicationconnector/v1alpha1"
	acClient "github.com/kyma-project/kyma/components/application-operator/pkg/client/clientset/versioned/typed/applicationconnector/v1alpha1"
	sourcesclientv1alpha1 "github.com/kyma-project/kyma/components/event-sources/client/generated/clientset/internalclientset/typed/sources/v1alpha1"
	"github.com/kyma-project/kyma/tests/end-to-end/external-solution-integration/pkg/helpers"
	"github.com/kyma-project/kyma/tests/end-to-end/external-solution-integration/pkg/step"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// CreateApplication is a step which creates new Application
type CreateApplication struct {
	applications     acClient.ApplicationInterface
	httpSources      sourcesclientv1alpha1.HTTPSourceInterface
	skipInstallation bool
	name             string
	accessLabel      string
	tenant           string
	group            string
}

var _ step.Step = &CreateApplication{}

// NewCreateApplication returns new CreateApplication
func NewCreateApplication(name, accessLabel string, skipInstallation bool, tenant, group string,
	applications acClient.ApplicationInterface, httpSourceClient sourcesclientv1alpha1.HTTPSourceInterface) *CreateApplication {
	return &CreateApplication{
		name:             name,
		applications:     applications,
		httpSources:      httpSourceClient,
		skipInstallation: skipInstallation,
		accessLabel:      accessLabel,
		tenant:           tenant,
		group:            group,
	}
}

// Name returns name name of the step
func (s *CreateApplication) Name() string {
	return fmt.Sprintf("Create application %s", s.name)
}

// Run executes the step
func (s *CreateApplication) Run() error {
	spec := acApi.ApplicationSpec{
		Services:         []acApi.Service{},
		AccessLabel:      s.accessLabel,
		SkipInstallation: s.skipInstallation,
		Tenant:           s.tenant,
		Group:            s.group,
	}

	dummyApp := &acApi.Application{
		TypeMeta:   v1.TypeMeta{Kind: "Application", APIVersion: acApi.SchemeGroupVersion.String()},
		ObjectMeta: v1.ObjectMeta{Name: s.name},
		Spec:       spec,
	}

	_, err := s.applications.Create(dummyApp)
	if err != nil {
		return err
	}

	return retry.Do(s.isApplicationReady, retry.Delay(time.Duration(200) * time.Millisecond))
}

func (s *CreateApplication) isApplicationReady() error {
	application, err := s.applications.Get(s.name, v1.GetOptions{})

	if err != nil {
		return err
	}

	if application.Status.InstallationStatus.Status == "DEPLOYED" {
		return errors.Errorf("unexpected installation status: %s", application.Status.InstallationStatus.Status)
	}

	//if s.httpSources != nil {
	//	httpSource, err := s.httpSources.Get(s.name, v1.GetOptions{})
	//	if err != nil {
	//		return err
	//	}
	//	fmt.Printf( "Installation Status: %s \n", httpSource.Status.IsReady())
	//	fmt.Printf( "HTTPSource Status: %s \n", httpSource.Status)
	//	if !httpSource.Status.IsReady() {
	//		return errors.Errorf("httpSource is not ready")
	//	}
	//}
	return nil
}

// Cleanup removes all resources that may possibly created by the step
func (s *CreateApplication) Cleanup() error {
	err := s.applications.Delete(s.name, &v1.DeleteOptions{})
	if err != nil {
		return err
	}

	return helpers.AwaitResourceDeleted(func() (interface{}, error) {
		return s.applications.Get(s.name, v1.GetOptions{})
	})
}
