package subject

import (
    "fmt"
    "github.com/awakari/metrics/model"
)

func Encode(src model.Subject) (dst Subject, err error) {
    switch src {
    case model.SubjectInterests:
        dst = Subject_Interests
    case model.SubjectPublishEvents:
        dst = Subject_PublishEvents
    default:
        err = fmt.Errorf(fmt.Sprintf("invalid subject: %s", src))
    }
    return
}
