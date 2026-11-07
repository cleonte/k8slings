# serviceaccounts1: a token that is mounted although it was turned off

Every Pod gets a **ServiceAccount token** mounted into its containers, unless automounting is switched off, so that a Pod that never talks to the API does not carry a credential that could be stolen. `automountServiceAccountToken: false` can be set on the **ServiceAccount** (a default for its Pods) and on the **Pod** (which wins when both say something).

The Pod `reporter` never talks to the API and uses the ServiceAccount `reporter`, where the token has been turned off. It still gets one. Find out why, and make the Pod run without a token.
