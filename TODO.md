

1. Confirmation: Is the Crdt behavior getting latest data or not when lounch(boot) time.
2. Structure: peerID, diff-data, VersionNomber, Timestamp


# CRDT

CRDTの仕様に乗っかる場合、TimeStamp, VersionNo. などは設計上基本必要ない

**Receiver**

1. .snap監視
2. .snap変更通知
  - Put/Delete LocalTree 更新

**Sender**
1. .snap 同期
2. LocalTree 更新


# IPFS

分散されているので、
最新データを保持しているPeerIDが存在しない。などを解消することができる、と思う。


