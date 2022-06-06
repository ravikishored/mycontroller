/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	intuitv1alpha1 "my.domain/mycontroller/api/v1alpha1"
)

// FlipperReconciler reconciles a Flipper object
type FlipperReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=intuit.my.domain,resources=flippers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=intuit.my.domain,resources=flippers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=intuit.my.domain,resources=flippers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Flipper object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *FlipperReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	// TODO(user): your logic here
	logger.Info("running control loop \n")
	flipper := &intuitv1alpha1.Flipper{}
	if err := r.Get(ctx, req.NamespacedName, flipper); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	fmt.Println("Interval", flipper.Spec.Interval)
	fmt.Println("Labels", flipper.Spec.Match.Labels)
	fmt.Println("Namespace", flipper.Spec.Match.Namespace)
	clientset := GetLocalClient(os.Getenv("KUBECONFIG_PATH"))

	deployments, err := clientset.AppsV1().Deployments(flipper.Spec.Match.Namespace).List(
		context.TODO(), v1.ListOptions{},
	)
	if err != nil {
		fmt.Println("Failed to fetch deployment")
	}

	var alldeployments []string
	for _, v := range deployments.Items {
		for labelKey, labelValue := range v.Spec.Selector.MatchLabels {
			for matchKey, matchValue := range flipper.Spec.Match.Labels {
				if labelKey == matchKey && labelValue == matchValue {
					alldeployments = append(alldeployments, v.ObjectMeta.Name)
				}
			}
		}
	}

	fmt.Println("Matching deployments: ", alldeployments)
	if len(alldeployments) > 0 {
		data := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().String())
		resultDeployment, err := clientset.AppsV1().Deployments("mesh").Patch(context.Background(),
			alldeployments[0], types.StrategicMergePatchType, []byte(data), v1.PatchOptions{FieldManager: "kubectl-rollout"})
		if err != nil {
			panic(err)
		}
		fmt.Print(resultDeployment)
	}

	return ctrl.Result{RequeueAfter: time.Duration(flipper.Spec.Interval) * time.Second}, nil
	// return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlipperReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&intuitv1alpha1.Flipper{}).
		Complete(r)
}

func GetLocalClient(kubeconfigpath string) *kubernetes.Clientset {
	fmt.Println("Kubeconfig path:", kubeconfigpath)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigpath)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return clientset
}
