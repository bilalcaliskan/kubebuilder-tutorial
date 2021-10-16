/*
Copyright 2021.

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

package v1

import (
	"github.com/robfig/cron"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// We’ll setup a logger for the webhooks.
var cronjoblog = logf.Log.WithName("cronjob-resource")

// SetupWebhookWithManager sets up the webhook with the manager which also manages controllers
func (r *CronJob) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

/*
Notice that we use kubebuilder markers to generate webhook manifests. This marker is responsible for generating a
mutating webhook manifest.

The meaning of each marker can be found on https://book.kubebuilder.io/reference/markers.html.
*/
//+kubebuilder:webhook:path=/mutate-batch-example-com-v1-cronjob,mutating=true,failurePolicy=fail,sideEffects=None,groups=batch.example.com,resources=cronjobs,verbs=create;update,versions=v1,name=mcronjob.kb.io,admissionReviewVersions={v1,v1beta1}

/*
We use the webhook.Defaulter interface to set defaults to our CRD. A webhook will automatically be served that calls this defaulting.
The Default method is expected to mutate the receiver, setting the defaults.
*/
var _ webhook.Defaulter = &CronJob{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CronJob) Default() {
	cronjoblog.Info("default", "name", r.Name)

	if r.Spec.ConcurrencyPolicy == "" {
		r.Spec.ConcurrencyPolicy = AllowConcurrent
	}

	if r.Spec.Suspend == nil {
		r.Spec.Suspend = new(bool)
	}

	if r.Spec.SuccessfulJobsHistoryLimit == nil {
		r.Spec.SuccessfulJobsHistoryLimit = new(int32)
		*r.Spec.SuccessfulJobsHistoryLimit = 3
	}

	if r.Spec.FailedJobsHistoryLimit == nil {
		r.Spec.FailedJobsHistoryLimit = new(int32)
		*r.Spec.FailedJobsHistoryLimit = 1
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// This marker is responsible for generating a validating webhook manifest.
//+kubebuilder:webhook:path=/validate-batch-example-com-v1-cronjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=batch.example.com,resources=cronjobs,verbs=create;update,versions=v1,name=vcronjob.kb.io,admissionReviewVersions={v1,v1beta1}

/*
To validate our CRD beyond what’s possible with declarative validation. Generally, declarative validation should be
sufficient, but sometimes more advanced use cases call for complex validation.

For instance, we’ll see below that we use this to validate a well-formed cron schedule without making up a long
regular expression.

If webhook.Validator interface is implemented, a webhook will automatically be served that calls the validation.

The ValidateCreate, ValidateUpdate and ValidateDelete methods are expected to validate that its receiver upon
creation, update and deletion respectively. We separate out ValidateCreate from ValidateUpdate to allow behavior
like making certain fields immutable, so that they can only be set on creation. ValidateDelete is also separated
from ValidateUpdate to allow different validation behavior on deletion. Here, however, we just use the same shared
validation for ValidateCreate and ValidateUpdate. And we do nothing in ValidateDelete, since we don’t need to
validate anything on deletion.
*/

var _ webhook.Validator = &CronJob{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CronJob) ValidateCreate() error {
	cronjoblog.Info("validate create", "name", r.Name)
	return r.validateCronJob()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CronJob) ValidateUpdate(old runtime.Object) error {
	cronjoblog.Info("validate update", "name", r.Name)
	return r.validateCronJob()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CronJob) ValidateDelete() error {
	cronjoblog.Info("validate delete", "name", r.Name)
	return nil
}

// validateCronJob validates the name and the spec of the CronJob.
func (r *CronJob) validateCronJob() error {
	var allErrs field.ErrorList
	if err := r.validateCronJobName(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateCronJobSpec(); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(schema.GroupKind{Group: "batch.example.com", Kind: "CronJob"}, r.Name, allErrs)
}

/*
Some fields are declaratively validated by OpenAPI schema. You can find kubebuilder validation markers (prefixed
with // +kubebuilder:validation) in the https://book.kubebuilder.io/cronjob-tutorial/api-design.html.
*/

/*
validateCronJobName validates the ObjectMeta.Name field of the object

Validating the length of a string field can be done declaratively by the validation schema.

But the ObjectMeta.Name field is defined in a shared package under the apimachinery repo, so we can’t
declaratively validate it using the validation schema.
*/
func (r *CronJob) validateCronJobName() *field.Error {
	if len(r.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-11 {
		/*
			The job name length is 63 character like all Kubernetes objects (which must fit in a DNS subdomain).
			The cronjob controller appends a 11-character suffix to the cronjob (`-$TIMESTAMP`) when creating
			a job. The job name length limit is 63 characters. Therefore cronjob names must have length <= 63-11=52. If
			we don't validate this here, then job creation will fail later.
		*/
		return field.Invalid(field.NewPath("metadata").Child("name"), r.Name, "must be no more than 52 characters")
	}
	return nil
}

// validateCronJobSpec validates the .spec of our CRD
func (r *CronJob) validateCronJobSpec() *field.Error {
	// The field helpers from the kubernetes API machinery help us return nicely structured validation errors.
	return validateScheduleFormat(
		r.Spec.Schedule,
		field.NewPath("spec").Child("schedule"))
}

// validateScheduleFormat validates the cron schedule is well-formatted.
func validateScheduleFormat(schedule string, fldPath *field.Path) *field.Error {
	if _, err := cron.ParseStandard(schedule); err != nil {
		return field.Invalid(fldPath, schedule, err.Error())
	}
	return nil
}
