package core

func isActive(active, name string) string {
	if active == name {
		return "bg-blue-50 text-blue-700 border-b-2 border-blue-600 font-medium"
	}
	return "text-slate-600 hover:text-slate-900"
}

func isSelected(active, name string) string {
	if active == name {
		return "true"
	}
	return "false"
}
