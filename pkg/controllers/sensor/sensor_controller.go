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

package sensor

import (
	"context"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "eventrigger.com/operator/pkg/api/v1"
)

const (
	// ControllerName is name of the controller
	ControllerName = "sensor-controller"

	finalizerName = ControllerName
)

// SensorReconciler reconciles a Sensor object
type SensorReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
	logger *zap.SugaredLogger
}

//+kubebuilder:rbac:groups=core.eventrigger.com,resources=sensors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.eventrigger.com,resources=sensors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.eventrigger.com,resources=sensors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Sensor object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *SensorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	sensor := &corev1.Sensor{}
	if err := r.Client.Get(ctx, req.NamespacedName, sensor); err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Warnw("WARNING: sensor not found", "request", req)
			return reconcile.Result{}, nil
		}
		r.logger.Errorw("unable to get sensor ctl", zap.Any("request", req), zap.Error(err))
		return ctrl.Result{}, err
	}
	log := r.logger.With("namespace", sensor.Namespace).With("sensor", sensor.Name)
	sensorCopy := sensor.DeepCopy()
	reconcileErr := r.reconcile(ctx, sensorCopy)
	if reconcileErr != nil {
		log.Errorw("reconcile error", zap.Error(reconcileErr))
	}

	if err := r.Client.Status().Update(ctx, sensorCopy); err != nil {
		return reconcile.Result{}, err
	}
	return ctrl.Result{}, nil
}

// reconcile does the real logic
func (r *SensorReconciler) reconcile(ctx context.Context, sensor *corev1.Sensor) error {
	log := r.logger.With("namespace", sensor.Namespace).With("sensor", sensor.Name)
	if !sensor.DeletionTimestamp.IsZero() {
		log.Info("deleting sensor")
		if controllerutil.ContainsFinalizer(sensor, finalizerName) {
			// Finalizer logic should be added here.
			controllerutil.RemoveFinalizer(sensor, finalizerName)
		}
		return nil
	}
	controllerutil.AddFinalizer(sensor, finalizerName)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SensorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Sensor{}).
		Complete(r)
}
