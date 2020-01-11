var websocket
$(document).ready(function () {
    let auth = getLocalStorage("authToken");
    let jsonData = {"authToken": auth};
    $.ajax({
        type: "POST",
        contentType: "application/json",
        dataType: "json",
        url: apiUrl + "/user/checkAuth",
        data: JSON.stringify(jsonData),
        success: function (result) {
            if (result.code == 0) {
                $("#chatroom-verified").css("display", "block");
                $("#chatroom-anonymous").css("display", "none");
                $("#nickName").text(result.data.userName);
                $("#nickName").css("display", "inline");
                $("#chatroom-login").css("display", "none");
            } else {
                $("#chatroom-verified").css("display", "none");
                $("#chatroom-anonymous").css("display", "block");
                $("#nickName").css("display", "none");
                $("#chatroom-login").css("display", "inline");
                $("#chatroom-logout").css("display", "none");
            }
        },
        error: function () {
            swal("sorry, exception!");
        }
    });
     websocket = new WebSocket(socketUrl );
    let data = {"authToken": getLocalStorage("authToken"), "roomId": 1};
    let data2 ={ver:1,op:3,seq:"123",body:{msg: "大家好", roomId: 1}}
    var timestamp = (new Date()).valueOf().toString()
    let data3 ={ver:1,op:2,seq:timestamp,body:{msg: "newgame", toUserId: 5}}

    //websocket onopen
    websocket.onopen = function (evt) {


        //getRoomInfo();
    };
    function sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms))
    }
    websocket.onmessage = function (evt) {
        let data = JSON.parse(evt.data);
        if (data.op == 3) {
            let userNameAndMsg = data.body.fromUserName + '(' + data.body.createTime + ')';
            let innerInfo = '<div class="item" >' +
                '<p class="nick guest j-nick " data-role="guest"></p>' +
                '<p class="text"></p>' +
                '</div>';
            $("#msg").append(innerInfo);
            $("#msg > div[class='item']:last > p[class='nick guest j-nick ']").text(userNameAndMsg);
            $("#msg > div[class='item']:last > p[class='text']:last").text(data.body.msg);
            $("#msg").animate({scrollTop: $("#msg").offset().top + 100000}, 1000);
        } else if (data.op == 4) {
            // get room user count
            $("#roomOnlineMemberNum").text(data.body.count);
        } else if (data.op == 5) {
            // get room user list
            $('#member_info').html("");
            let innerInfoArr = [];
            for (let k in data.body.roomUserInfo) {
                let item = '<div class="item" data-id="' + k + '"><div class="avatar"><img src="/static/chat_head.jpg"> </div> <div class="nick">' + data.body.roomUserInfo[k] + '</div> </div>';
                innerInfoArr.push(item)
            }
            $('#member_info').html(innerInfoArr.join(""));
            $("#roomOnlineMemberNum").text(data.body.count);
        }
    };
});

function getRoomInfo() {
    let jsonData = {roomId: 1, authToken: getLocalStorage("authToken")};
    $.ajax({
        type: "POST",
        dataType: "json",
        url: apiUrl + "/push/getRoomInfo",
        data: JSON.stringify(jsonData),
        success: function (result) {
            if (result.code != 0) {
                //swal("request error，please try again later！");
            }
        },
        error: function () {
            swal("sorry, exception!");
        }
    });
}


function getRoomUserCount() {
    let jsonData = {roomId: 1, authToken: getLocalStorage("authToken")};
    $.ajax({
        type: "POST",
        dataType: "json",
        url: apiUrl + "/push/count",
        data: JSON.stringify(jsonData),
        success: function (result) {
            if (result.code != 0) {
                swal("request error，please login!");
            }
        },
        error: function () {
            swal("sorry, exception!");
        }
    });
}

$("#editText").keypress(function (e) {
    if (e.which == 13) {
        send();
    }
});

function send() {
    $("#tab_chat").click();
    $("#msg").animate({scrollTop: $("#msg").offset().top + 100000}, 1000);
    let msg2 = document.getElementById('editText').value;
    if (msg2 == "") {
        swal("send msg is empty!");
        return
    }
    document.getElementById('editText').value = '';
    var timestamp = (new Date()).valueOf().toString()
    let jsonData = `{"ver":1,"op":2,"seq":"`+timestamp+`","body":{"msg":"`+msg2+`","toUserId":5}}`
    /**$.ajax({
        type: "POST",
        dataType: "json",
        url: apiUrl + "/push/pushRoom",
        data: JSON.stringify(jsonData),
        success: function (result) {
            if (result.code == 0) {
                // send ok
            } else {
                swal("please login or register account!");
                window.location.href = "/register.html";
            }
        },
        error: function () {
            swal("sorry, exception！");
        }
    });**/
    websocket.send(jsonData);
}


function logout() {
    let jsonData = {authToken: getLocalStorage("authToken")};
    $.ajax({
        type: "POST",
        dataType: "json",
        url: apiUrl + "/user/logout",
        data: JSON.stringify(jsonData),
        success: function (result) {
            if (result.code == 0) {
                window.location.href = "/login.html";
            } else {
                swal("request error，please login！");
            }
        },
        error: function () {
            swal("sorry, exception！");
        }
    });
}

function changeTab(type) {
    if (type == "chat") {
        document.getElementById("tab_chat").className = "crt j-tab";
        document.getElementById("tab_member").className = "j-tab";
        document.getElementById("msg").className = "chat j-pannel j-chat";
        document.getElementById("member_list").className = "member j-pannel hide";
    } else {
        document.getElementById("tab_chat").className = "j-tab";
        document.getElementById("tab_member").className = "crt j-tab";
        document.getElementById("member_list").className = "member j-pannel";
        document.getElementById("msg").className = "chat j-pannel j-chat hide";
        getRoomInfo();
        getRoomUserCount();
    }
}