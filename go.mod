module github.com/kubescape/operator

go 1.21.3

require (
	github.com/armosec/armoapi-go v0.0.330
	github.com/armosec/cluster-notifier-api-go v0.0.5
	github.com/armosec/registryx v0.0.15
	github.com/armosec/utils-go v0.0.57
	github.com/armosec/utils-k8s-go v0.0.26
	github.com/aws/aws-sdk-go v1.50.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/deckarep/golang-set/v2 v2.6.0
	github.com/distribution/reference v0.5.0
	github.com/docker/docker v25.0.5+incompatible
	github.com/go-openapi/runtime v0.27.1
	github.com/google/go-containerregistry v0.19.0
	github.com/google/uuid v1.6.0
	github.com/goradd/maps v0.1.5
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/kubescape/backend v0.0.19
	github.com/kubescape/go-logger v0.0.22
	github.com/kubescape/k8s-interface v0.0.161
	github.com/kubescape/opa-utils v0.0.278
	github.com/kubescape/storage v0.0.62
	github.com/mitchellh/mapstructure v1.5.0
	github.com/panjf2000/ants/v2 v2.8.2
	github.com/spf13/viper v1.18.2
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.44.0
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/trace v1.22.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
	gotest.tools/v3 v3.5.0
	k8s.io/api v0.29.2
	k8s.io/apimachinery v0.29.2
	k8s.io/client-go v0.29.2
	k8s.io/utils v0.0.0-20231121161247-cf03d44ff3cf
	sigs.k8s.io/yaml v1.4.0
)

require (
	github.com/IBM/sarama v1.42.1 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/eapache/go-resiliency v1.4.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.4.0 // indirect
	github.com/elastic/go-elasticsearch/v8 v8.12.0 // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/gocql/gocql v1.6.0 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/montanaflynn/stats v0.7.0 // indirect
	github.com/streadway/amqp v1.1.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	go.etcd.io/etcd/api/v3 v3.5.11 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.11 // indirect
	go.etcd.io/etcd/client/v3 v3.5.11 // indirect
)

require (
	cloud.google.com/go/compute v1.23.3 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/container v1.29.0 // indirect
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20230811130428-ced1acdcaa24 // indirect
	github.com/AdamKorcz/go-118-fuzz-build v0.0.0-20230306123547-8075edf89bb0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.11.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.6.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.8.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2 v2.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2 v2.4.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Microsoft/hcsshim v0.11.4 // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/acobaugh/osrelease v0.1.0 // indirect
	github.com/adrg/xdg v0.4.0 // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/anchore/clio v0.0.0-20231016125544-c98a83e1c7fc // indirect
	github.com/anchore/fangs v0.0.0-20231201140849-5075d28d6d8b // indirect
	github.com/anchore/go-logger v0.0.0-20230725134548-c21dafa1ec5a // indirect
	github.com/anchore/packageurl-go v0.1.1-0.20230104203445-02e0a6721501 // indirect
	github.com/anchore/stereoscope v0.0.1 // indirect
	github.com/anchore/syft v0.101.1 // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/armosec/gojay v1.2.17 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2 v1.24.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.26.6 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.16.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.20.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/eks v1.28.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/iam v1.21.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.7 // indirect
	github.com/aws/smithy-go v1.19.0 // indirect
	github.com/becheran/wildmatch-go v1.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.6.1 // indirect
	github.com/briandowns/spinner v1.23.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/containerd/cgroups v1.1.0 // indirect
	github.com/containerd/containerd v1.7.12 // indirect
	github.com/containerd/continuity v0.4.2 // indirect
	github.com/containerd/fifo v1.1.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.14.3 // indirect
	github.com/containerd/ttrpc v1.2.2 // indirect
	github.com/containerd/typeurl/v2 v2.1.1 // indirect
	github.com/coreos/go-oidc v2.2.1+incompatible // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/cli v24.0.7+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.1 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dsnet/compress v0.0.2-0.20210315054119-f66993602bf5 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch v5.7.0+incompatible // indirect
	github.com/facebookincubator/nvdtools v0.1.5 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/fgprof v0.9.3 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/github/go-spdx/v2 v2.2.0 // indirect
	github.com/go-gota/gota v0.12.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.22.0 // indirect
	github.com/go-openapi/errors v0.21.0 // indirect
	github.com/go-openapi/jsonpointer v0.20.2 // indirect
	github.com/go-openapi/jsonreference v0.20.4 // indirect
	github.com/go-openapi/loads v0.21.5 // indirect
	github.com/go-openapi/spec v0.20.13 // indirect
	github.com/go-openapi/strfmt v0.22.0 // indirect
	github.com/go-openapi/swag v0.22.9 // indirect
	github.com/go-openapi/validate v0.22.4 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/glog v1.1.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20231023181126-ff6d637d2a7b // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/kubescape/kubescape-network-scanner v0.0.15
	github.com/kubescape/kubescape/v3 v3.0.4 // indirect
	github.com/kubescape/rbac-utils v0.0.21-0.20230806101615-07e36f555520 // indirect
	github.com/kubescape/regolibrary v1.0.315 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mholt/archiver/v3 v3.5.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/signal v0.7.0 // indirect
	github.com/moby/sys/user v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/olvrng/ujson v1.1.0 // indirect
	github.com/open-policy-agent/opa v0.61.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc5 // indirect
	github.com/opencontainers/runtime-spec v1.1.0 // indirect
	github.com/opencontainers/selinux v1.11.0 // indirect
	github.com/pborman/indent v1.2.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/profile v1.7.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/pquerna/cachecontrol v0.2.0 // indirect
	github.com/prometheus/client_golang v1.18.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/scylladb/go-set v1.0.3-0.20200225121959-cc7b2070d91e // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/cobra v1.8.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stripe/stripe-go/v74 v74.30.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/sylabs/sif/v2 v2.11.5 // indirect
	github.com/sylabs/squashfs v0.6.1 // indirect
	github.com/tchap/go-patricia/v2 v2.3.1 // indirect
	github.com/therootcompany/xz v1.0.1 // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/uptrace/opentelemetry-go-extra/otelutil v0.2.3 // indirect
	github.com/uptrace/opentelemetry-go-extra/otelzap v0.2.3 // indirect
	github.com/uptrace/uptrace-go v1.21.0 // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	github.com/wagoodman/go-partybus v0.0.0-20230516145632-8ccac152c651 // indirect
	github.com/wagoodman/go-progress v0.0.0-20230925121702-07e42b3cdba0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	github.com/yashtewari/glob-intersection v0.2.0 // indirect
	go.mongodb.org/mongo-driver v1.13.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.47.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.47.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.46.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.22.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.22.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.21.0 // indirect
	go.opentelemetry.io/otel/metric v1.22.0 // indirect
	go.opentelemetry.io/otel/sdk v1.22.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.21.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/oauth2 v0.16.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/term v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	gonum.org/v1/gonum v0.9.1 // indirect
	google.golang.org/api v0.159.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20240116215550-a9fa1716bcac // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240116215550-a9fa1716bcac // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240116215550-a9fa1716bcac // indirect
	google.golang.org/grpc v1.61.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.110.1 // indirect
	k8s.io/kube-openapi v0.0.0-20231113174909-778a5567bc1e // indirect
	sigs.k8s.io/controller-runtime v0.16.3 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)

replace github.com/project-copacetic/copacetic => github.com/anubhav06/copacetic v0.0.0-20230821175613-0a7915a62e10
