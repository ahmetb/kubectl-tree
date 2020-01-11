package main

var namespace *string

func getNamespace() string {
	if namespace != nil {
		return *namespace
	}
	ns := *cf.Namespace
	if ns == "" {
		clientConfig := cf.ToRawKubeConfigLoader()
		defaultNamespace, _, err := clientConfig.Namespace()
		if err != nil {
			defaultNamespace = "default"
		}
		ns = defaultNamespace
	}
	namespace = &ns
	return ns
}
