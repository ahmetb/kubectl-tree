package main

func getNamespace() string {
	ns := *cf.Namespace
	if ns == "" {
		clientConfig := cf.ToRawKubeConfigLoader()
		defaultNamespace, _, err := clientConfig.Namespace()
		if err != nil {
			defaultNamespace = "default"
		}
		ns = defaultNamespace
	}
	return ns
}
