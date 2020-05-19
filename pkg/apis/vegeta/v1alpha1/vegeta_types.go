package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AttackSpec describes the attack parameters.
type AttackSpec struct {
	// Body specifies the requests body.
	//
	// +optional
	Body string `json:"body,omitempty"`
	// Cert specifies the TLS client PEM encoded certificate file.
	//
	// +optional
	Cert string `json:"cert,omitempty"`
	// Chunked is used to send body with chunked transfer encoding.
	//
	// +optional
	Chunked bool `json:"chunked,omitempty"`
	// Connections referes to the max open idle connections per target host (default 10000).
	//
	// +optional
	Connections int `json:"connections,omitempty"`
	// Duration defines the duration of the test [0 = forever].
	//
	// +optional
	Duration string `json:"duration,omitempty"`
	// H2C is used to send HTTP/2 requests without TLS encryption.
	//
	// +optional
	H2C bool `json:"h2c,omitempty"`
	// Header defines the request header.
	//
	// +optional
	Header string `json:"header,omitempty"`
	// HTTP2 sends HTTP/2 requests when supported by the server (default true).
	//
	// +optional
	HTTP2 bool `json:"http2,omitempty"`
	// Insecure is used to ingore valid server TLS certificates.
	//
	// +optional
	Insecure bool `json:"insecure,omitempty"`
	// KeepAlive is used to have persistent connections (default true).
	//
	// +optional
	KeepAlive bool `json:"keepAlive,omitempty"`
	// Key specified the TLS client PEM encoded private key file.
	//
	// +optional
	Key string `json:"key,omitempty"`
	// LAddr defines the local IP address (default 0.0.0.0)
	//
	// +optional
	LAddr string `json:"laddr,omitempty"`
	// Lazy specifies whether to read the input targets lazily instead of eagerly.
	//
	// +optional
	Lazy bool `json:"lazy,omitempty"`
	// MaxBody specifies the maximum number of bytes to capture from response bodies. [-1 = no limit] (default -1).
	//
	// +optional
	MaxBody uint `json:"maxBody,omitempty"`
	// MaxWorkers specifies the Maximum number of workers (default 18446744073709551615).
	//
	// +optional
	MaxWorkers uint `json:"maxWorkers,omitempty"`
	// Name defines the attack name.
	//
	// +optional
	Name string `json:"name,omitempty"`
	// Output defines the output file (default "stdout").
	//
	// +optional
	Output string `json:"output,omitempty"`
	// ProxyHeader specifies the Proxy CONNECT header.
	//
	// +optional
	ProxyHeader string `json:"proxyHeader,omitempty"`
	// Rate defines the umber of requests per time unit [0 = infinity] (default 50/1s).
	//
	// +optional
	Rate string `json:"rate,omitempty"`
	// Redirects specifies the number of redirects to follow. -1 will not follow but marks as success (default 10).
	//
	// +optional
	Redirects int `json:"redirects,omitempty"`
	// Resolvers is the list of addresses (ip:port) to use for DNS resolution. Disables use of local system DNS. (comma separated list).
	//
	// +optional
	Resolvers string `json:"resolvers,omitempty"`
	// RootCerts are the TLS root certificate files (comma separated list).
	//
	// +optional
	RootCerts string `json:"rootcerts,omitempty"`
	// Targets specifies the targets file (default "stdin").
	//
	// +optional
	Targets string `json:"targets,omitempty"`
	// Timeout defines the requests timeout (default 30s).
	//
	// +optional
	Timeout string `json:"timeout,omitempty"`
	// UnixSocket specifies a Unix socket to connect to. This overrides the host address in target URLs.
	//
	// +optional
	UnixSocket string `json:"unixSocket,omitempty"`
	// Workers defines the initial number of workers (default 10).
	//
	// +optional
	Workers uint `json:"workers,omitempty"`

	// Report defines the reporting parameters.
	//
	// +optional
	Report *ReportSpec `json:"report,omitempty"`
}

// ReportSpec describes the report parameters.
type ReportSpec struct {
	// Buckets defines the histogram buckets, e.g.: "[0,1ms,10ms]".
	//
	// +optional
	Buckets string `json:"buckets,omitempty"`
	// Every defines the report interval.
	//
	// +optional
	Every string `json:"every,omitempty"`
	// Output defines the output file (default "stdout").
	//
	// +optional
	Output string `json:"output,omitempty"`
	// Type defines the report type to generate [text, json, hist[buckets], hdrplot] (default "text").
	//
	// +optional
	Type string `json:"type,omitempty"`
}

// BlobStorageSpec defines the cloud provider for storing the report in a bucket.
type BlobStorageSpec struct {
	// Provider specifies the cloud provider that will be used.
	//
	// +required
	Provider string `json:"provider"`
	// Name refers to the bucket name.
	//
	// +required
	Name string `json:"name"`
	// Env refers to the environment variables used for connecting to the cloud providers.
	//
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// VegetaSpec defines the desired state of Vegeta
type VegetaSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Target refers to the target endpoint for the load testing.
	// For multiple targets use the Targets fields in the the Attack Spec.
	//
	// +required
	Target string `json:"target"`
	// Attack specifies the attack parameters.
	//
	// +required
	Attack *AttackSpec `json:"attack"`

	// BlogStorageSpec refers to the cloud storage provider parameters for the bucket.
	//
	// +optional
	BlobStorage *BlobStorageSpec `json:"blobStorage,omitempty"`

	// Resources specifies the resource limits of the container.
	//
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// VegetaStatus defines the observed state of Vegeta
type VegetaStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Vegeta is the Schema for the vegeta API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=vegeta,scope=Namespaced
type Vegeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VegetaSpec   `json:"spec,omitempty"`
	Status VegetaStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VegetaList contains a list of Vegeta
type VegetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Vegeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Vegeta{}, &VegetaList{})
}
