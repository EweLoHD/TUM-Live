import { NewChatMessage } from "./NewChatMessage";
import { ChatUserList } from "./ChatUserList";
import { EmojiList } from "./EmojiList";
import { Poll } from "./Poll";

export class Chat {
    readonly userId: number;
    readonly userName: string;
    readonly admin: boolean;
    readonly streamId: number;

    orderByLikes: boolean;
    disconnected: boolean;
    current: NewChatMessage;
    messages: ChatMessage[];
    users: ChatUserList;
    emojis: EmojiList;
    poll: Poll;

    preprocessors: ((m: ChatMessage) => ChatMessage)[] = [
        (m: ChatMessage) => {
            if (m.addressedTo.find((uId) => uId === this.userId) !== undefined) {
                m.message = m.message.replaceAll(
                    "@" + this.userName,
                    "<span class = 'text-sky-800 bg-sky-200 text-xs dark:text-indigo-200 dark:bg-indigo-800 p-1 rounded'>" +
                        "@" +
                        this.userName +
                        "</span>",
                );
            }
            return m;
        },
    ];

    constructor(isAdminOfCourse: boolean, streamId: number, userId: number, userName: string) {
        this.orderByLikes = false;
        this.disconnected = false;
        this.current = new NewChatMessage();
        this.admin = isAdminOfCourse;
        this.users = new ChatUserList(streamId);
        this.emojis = new EmojiList();
        this.messages = [];
        this.streamId = streamId;
        this.userId = userId;
        this.userName = userName;
        this.poll = new Poll(streamId);
    }

    async loadMessages() {
        this.messages = [];
        fetchMessages(this.streamId).then((messages) => {
            messages.forEach((m) => this.addMessage(m));
        });
    }

    sortMessages() {
        this.messages.sort((m1, m2) => {
            if (this.orderByLikes) {
                // @ts-ignore
                if (m1.likes === m2.likes) {
                    // @ts-ignore
                    return m2.id - m1.id; // same amount of likes -> newer messages up
                }
                // @ts-ignore
                return m2.likes - m1.likes; // more likes -> up
            } else {
                // @ts-ignore
                return m1.ID < m2.ID ? -1 : 1; // newest messages last
            }
        });
    }

    onMessage(e) {
        this.addMessage(e.detail);
    }

    onDelete(e) {
        // @ts-ignore
        this.messages.find((m) => m.ID === e.detail.delete).deleted = true;
    }

    onLike(e) {
        // @ts-ignore
        this.messages.find((m) => m.ID === e.detail.likes).likes = e.detail.num;
    }

    onResolve(e) {
        // @ts-ignore
        this.messages.find((m) => m.ID === e.detail.resolve).resolved = true;
    }

    onReply(e) {
        // @ts-ignore
        this.messages.find((m) => m.ID === e.detail.replyTo.Int64).replies.push(e.detail);
    }

    onNewPoll(e) {
        if (!this.current.anonymous) {
            this.poll.result = null;
            this.poll.activePoll = { ...e.detail, selected: null };
        }
    }

    onPollOptionVotesUpdate(e) {
        this.poll.updateVotes(e.detail);
    }

    onPollOptionResult(e) {
        this.poll.activePoll = null;
        this.poll.result = e.detail;
    }

    onSubmit() {
        if (this.emojis.isValid()) {
            window.dispatchEvent(new CustomEvent("chatenter"));
        } else if (this.users.isValid()) {
            // @ts-ignore
            this.current.addAddressee(this.users.getSelected());
            this.users.clear();
        } else {
            this.current.send();
        }
    }

    onInputKeyUp(e) {
        if (this.emojis.isValid()) {
            let event = "";
            switch (e.keyCode) {
                case 9: {
                    event = "emojiselectiontriggered";
                    break;
                }
                case 38: {
                    event = "emojiarrowup";
                    break;
                }
                case 40: {
                    event = "emojiarrowdown";
                    break;
                }
                default: {
                    this.emojis.getEmojisForMessage(e.target.value, e.target.selectionStart);
                    return;
                }
            }
            window.dispatchEvent(new CustomEvent(event));
        } else if (this.users.isValid()) {
            switch (e.keyCode) {
                case 38: /* UP */ {
                    this.users.prev();
                    break;
                }
                case 40: /* DOWN */ {
                    this.users.next();
                    break;
                }
                default: {
                    this.users.filterUsers(e.target.value, e.target.selectionStart);
                    return;
                }
            }
        } else {
            this.users.filterUsers(e.target.value, e.target.selectionStart);
            this.emojis.getEmojisForMessage(e.target.value, e.target.selectionStart);
        }
    }

    getInputPlaceHolder(): string {
        if (this.disconnected) {
            return "Reconnecting to chat...";
        }
        if (this.current.replyTo === 0) {
            return "Send a message";
        } else {
            return "Reply [escape to cancel]";
        }
    }

    private addMessage(m: ChatMessage) {
        this.preprocessors.forEach((f) => (m = f(m)));
        this.messages.push(m);
    }
}

export async function fetchMessages(streamId: number): Promise<ChatMessage[]> {
    return await fetch("/api/chat/" + streamId + "/messages")
        .then((res) => res.json())
        .then((messages) => {
            return messages;
        });
}

type ChatMessage = {
    ID: number;
    admin: boolean;

    message: string;
    name: string;
    color: string;

    liked: false;
    likes: number;

    replies: object[];
    replyTo: object; // e.g.{Int64:0, Valid:false}

    addressedTo: number[];
    resolved: false;
    visible: true;

    CreatedAt: string;
    DeletedAt: string;
    UpdatedAt: string;
};
