## consensus算法的学习&实现

### 参考资料
[raft作者讲解paxos＆raft](https://ramcloud.stanford.edu/~ongaro/userstudy/)

### basic paxos

> basic　paxos是纯粹的共识算法，其目的是解决一个问题：让所有进程都认可一个值

#### 实现
系统中有多个proposer 多个acceptor.proposer发出提案，acceptor接收提案．为了解决问题，需要
1. 让proposer知道目前是否有已经被选中的值
2. 老的提案不能覆盖新的提案

为了做到这两点，所以引入两阶段提交
1. prepare1，proposer将一个独一无二的number广播给acceptor,acceptor接收到以后如果这个number大于接收过最大的number，将记录下来，并且返回已经接收的提案和接收提案的number(如果没有返回nil)
2. prepare2,　proposer收到大部分的回应以后．如果回应中包含已经存在的提案，那么选中acceptNumber最大的提案 并覆盖自己的提案
3. commit1 proposer将(number，提案)广播给acceptor,accetpor接收到以后如果number>=接收过最大的提案，那么将number设置为acceptNumber和minNumber(接收过最大number)　并选中提案，返回minNumber
4. commit2 proposer收到大部分的响应后，如果回应中的minNumber> number那么说明提案失败，需要再次propose，反之则说明提案已经通过