# File used to generate the demo GIF of `k8slings run`.
#
# Needs vhs (https://github.com/charmbracelet/vhs) and a k8slings built from this
# checkout on the PATH:
#   go install ./k8slings && vhs demo.tape

Output misc/demo.gif

Set Shell "bash"
Set FontSize 20
Set Width 1200
Set Height 420
Set Padding 15

Hide
Type "cd $(mktemp -d) && k8slings init > /dev/null && cd k8slings-exercises && clear"
Enter
Show

Type "k8slings run pods1"
Enter

Sleep 6s
