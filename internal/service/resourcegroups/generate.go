//go:generate go run ../../generate/tags/main.go -ListTags -ListTagsOp=GetTags -ListTagsInIDElem=Arn -ServiceTagsMap -TagOp=Tag -TagInIDElem=Arn -UntagOp=Untag -UntagInTagsElem=Keys -UpdateTags
//go:generate go run ../../generate/servicepackage/main.go
// ONLY generate directives and package declaration! Do not add anything else to this file.

package resourcegroups
