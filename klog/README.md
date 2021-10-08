Sample Go app that uses klog library

This sample project is for showing some features of [klog](https://github.com/kubernetes/klog) library

Leaving the line `flag.Set("v", "3")` as is and running the app, it will output the following logs:
```text
I1008 10:30:50.554400   17392 main.go:29] TestName "msg"="Hello," "pod"="192.168.0.1" "World"=null
I1008 10:30:50.554573   17392 main.go:37] TestName "msg"="hello" "pod"="192.168.0.1" "val1"=1 "val2"={"k":1}
E1008 10:30:50.554612   17392 main.go:47] TestName "msg"="uh oh" "error"=null "pod"="192.168.0.1" "reasons"=[0.1,0.11,3.14] "trouble"=true
E1008 10:30:50.554623   17392 main.go:48] TestName "msg"="goodbye" "error"="an error occurred" "pod"="192.168.0.1" "code"=-1
I1008 10:30:50.554632   17392 main.go:52] TestName "msg"="New level log" "container"="4" "pod"="192.168.0.1"
I1008 10:30:50.554643   17392 main.go:53] TestName "msg"="Testing with keys values" "container"="4" "pod"="192.168.0.1" "anotherkey"="anothervalue" "some key"="some value"
```
in other words, it will output all Error lines, and any Info lines that is of verbosity level 3 and lower.

If we change the line `flag.Set("v", "3")` to `flag.Set("v", "4")`, then it will output the following logs:
```text
I1008 10:33:49.386312   17447 main.go:29] TestName "msg"="Hello," "pod"="192.168.0.1" "World"=null
I1008 10:33:49.386498   17447 main.go:37] TestName "msg"="hello" "pod"="192.168.0.1" "val1"=1 "val2"={"k":1}
I1008 10:33:49.386510   17447 main.go:40] TestName "msg"="nice to meet you" "pod"="192.168.0.1"
I1008 10:33:49.386518   17447 main.go:43] TestName "msg"="nice to meet you too" "pod"="192.168.0.1"
E1008 10:33:49.386549   17447 main.go:47] TestName "msg"="uh oh" "error"=null "pod"="192.168.0.1" "reasons"=[0.1,0.11,3.14] "trouble"=true
E1008 10:33:49.386566   17447 main.go:48] TestName "msg"="goodbye" "error"="an error occurred" "pod"="192.168.0.1" "code"=-1
I1008 10:33:49.386579   17447 main.go:52] TestName "msg"="New level log" "container"="4" "pod"="192.168.0.1"
I1008 10:33:49.386589   17447 main.go:53] TestName "msg"="Testing with keys values" "container"="4" "pod"="192.168.0.1" "anotherkey"="anothervalue" "some key"="some value"
```

If we change the line `flag.Set("v", "3")` to `flag.Set("v", "2")`, then it will output the following logs:
```text
I1008 10:34:42.769309   17474 main.go:29] TestName "msg"="Hello," "pod"="192.168.0.1" "World"=null
I1008 10:34:42.769545   17474 main.go:37] TestName "msg"="hello" "pod"="192.168.0.1" "val1"=1 "val2"={"k":1}
E1008 10:34:42.769589   17474 main.go:47] TestName "msg"="uh oh" "error"=null "pod"="192.168.0.1" "reasons"=[0.1,0.11,3.14] "trouble"=true
E1008 10:34:42.769598   17474 main.go:48] TestName "msg"="goodbye" "error"="an error occurred" "pod"="192.168.0.1" "code"=-1
```

Other interesting things of note:

"`WithName` adds a new element to the logger's name.
Successive calls with WithName continue to append
suffixes to the logger's name.  It's strongly recommended
that name segments contain only letters, digits, and hyphens
(see the package documentation for more information)."

In the sample code, the logger's name is "TestName".