$(document).ready(function() {

    DEFAULT_COOKIE_EXPIRE_TIME = 30;

    uname = '';
    session = '';
    uid = 0;
    currentVideo = null;
    listedVideos = null;

    session = getCookie('sessionid');
    uname = getCookie('username');

    initPage(function() {
        if (listedVideos !== null) {
            currentVideo = listedVideos[0];
            selectVideo(listedVideos[0]['id']);
        }

        $(".video-item").click(function() {
            var self = this.id
            listedVideos.forEach(function(item, index) {
                if (item['id'] === self) {
                    currentVideo = item;
                    return
                }
            });

            selectVideo(self);
        });

        $("#items").on('click', '.del-video-button', function(e) {
        e.stopPropagation(); // 防止冒泡触发播放视频
        // 你的 id 可能是 'del-VID'，所以需要 substring
        var vid = this.id.substring(4); 
        deleteVideo(vid, function(res, err) {
            if (err !== null) {
                popupErrorMsg("encounter an error when try to delete video: " + vid);
                return;
            }
            popupNotificationMsg("Successfully deleted video: " + vid)
            // 刷新当前页面（或者重新触发一次加载列表函数）
            location.reload();
        });
    });

        $("#submit-comment").on('click', function() {
            var content = $("#comments-input").val();
            postComment(currentVideo['id'], content, function(res, err) {
                if (err !== null) {
                    popupErrorMsg("encounter and error when try to post a comment: " + content);
                    return;
                }

                if (res === "ok") {
                    popupNotificationMsg("New comment posted")
                    $("#comments-input").val("");

                    refreshComments(currentVideo['id']);
                }
            });
        });
    });

    // 1. 绑定 Lobby 点击事件
    $("#lobby-link").on('click', function(e) {
        e.preventDefault();
        // 样式切换
        $(".topnav a").removeClass("active");
        $(this).addClass("active");

        listLobbyVideos(function(res, err) {
            if (err != null) {
                popupErrorMsg('Error loading lobby videos');
                return;
            }
            var obj = JSON.parse(res);
            renderVideoList(obj['videos']);
        });
    });

    // 2. 绑定 My Videos 点击事件
    $("#myvideos-link").on('click', function(e) {
        e.preventDefault();
        // 样式切换
        $(".topnav a").removeClass("active");
        $(this).addClass("active");

        listAllUserVideos(function(res, err) {
            if (err != null) {
                popupErrorMsg('Error loading my videos');
                return;
            }
            var obj = JSON.parse(res);
            renderVideoList(obj['videos']);
        });
    });

    // home page event registry
    $("#regbtn").on('click', function(e) {
        $("#regbtn").text('Loading...')
        e.preventDefault()
        registerUser(function(res, err) {
            $('#regbtn').text("Register")
            if (err != null) {
                popupErrorMsg('encounter an error, pls check your username or password');
                return;
            }

            popupNotificationMsg("Register successfully! Please Sign In.");

            // 切换到登录界面 (隐藏注册表单，显示登录表单)
            $("#regsubmit").hide();
            $("#signinsubmit").show();
        });
    });

    $("#signinbtn").on('click', function(e) {

        $("#signinbtn").text('Loading...')
        e.preventDefault();
        signinUser(function(res, err) {
            if (err != null) {
                $('#signinbtn').text("Sign In");
                //window.alert('encounter an error, pls check your username or password')
                popupErrorMsg('encounter an error, pls check your username or password');
                return;
            }

            var obj = JSON.parse(res);
            setCookie("sessionid", obj["session_id"], DEFAULT_COOKIE_EXPIRE_TIME);
            setCookie("username", uname, DEFAULT_COOKIE_EXPIRE_TIME);
            window.console.log("loginand set Cookie")
            //$("#signinsubmit").submit();
            //document.getElementById("signinsubmit").submit()
            window.location.href = "/userhome";
        });
    });

    $("#signinhref").on('click', function() {
        $("#regsubmit").hide();
        $("#signinsubmit").show();
    });

    $("#registerhref").on('click', function() {
        $("#regsubmit").show();
        $("#signinsubmit").hide();
    });

    // userhome event register
    $("#upload").on('click', function() {
        $("#uploadvideomodal").show();

    });


    $("#uploadform").on('submit', function(e) {
        e.preventDefault()
        var vname = $('#vname').val();

        createVideo(vname, function(res, err) {
            if (err != null ) {
                //window.alert('encounter an error when try to create video');
                popupErrorMsg('encounter an error when try to create video');
                return;
            }

            var obj = JSON.parse(res);
            var formData = new FormData();
            formData.append('file', $('#inputFile')[0].files[0]);

            $.ajax({
                url : 'http://' + window.location.hostname + ':8080/videos/upload/' + obj['id'],
                type : 'POST',
                data : formData,
                headers: {
                    'X-Session-Id': getCookie("sessionid")
                },
                crossDomain: true,
                processData: false,  // tell jQuery not to process the data
                contentType: false,  // tell jQuery not to set contentType
                success : function(data) {
                    console.log(data);
                    $('#uploadvideomodal').hide();
                    location.reload();
                    //window.alert("hoa");
                },
                complete: function(xhr, textStatus) {
                    if (xhr.status === 204) {
                        window.alert("finish")
                        return;
                    }
                    if (xhr.status === 400) {
                        $("#uploadvideomodal").hide();
                        popupErrorMsg('file is too big');
                        return;
                    }
                }

            });
        });
    });

    $(".close").on('click', function() {
        $("#uploadvideomodal").hide();
    });

    $("#logout").on('click', function() {
        var data={
            'url': 'http://' + window.location.hostname + ':8000/user/' + uname + '/logout',
            'method': 'POST',
            'req_body': ''
        };

        $.ajax({
            url: apiUrl,
            type: 'post',
            data: JSON.stringify(data),
            headers: {
                'X-Session-Id': getCookie("sessionid") // 必须带上 SID 才能找到要删哪个
            },
            // 无论后端成功与否，前端都要清除 cookie 并跳转
            complete: function() {
                setCookie("sessionid", "", -1);
                setCookie("username", "", -1);
                window.location.href = "/";
            }
        });
    });

    /* $(".video-item").click(function () {
        var url = 'http://' + window.location.hostname + ':9090/videos/'+ this.id
        var video = $("#curr-video");
        video[0].attr('src', url);
        video.load();
    }); */

    // 修复：使用事件委托绑定点击事件
    // 即使 .video-item 是后续 ajax 动态添加的，这个绑定依然有效
    $("#items").on('click', '.video-item', function() {
        var selfId = this.id;
        
        // 更新 currentVideo 对象
        listedVideos.forEach(function(item, index) {
            if (item['id'] === selfId) {
                currentVideo = item;
                return;
            }
        });

        // 选中播放
        selectVideo(selfId);
    });
});

function initPage(callback) {
    /* getUserId(function(res, err) {
        if (err != null) {
            window.alert("Encountered error when loading user id");
            return;
        }

        var obj = JSON.parse(res);
        uid = obj['id'];
        //window.alert(obj['id']);
        listAllVideos(function(res, err) {
            if (err != null) {
                //window.alert('encounter an error, pls check your username or password');
                popupErrorMsg('encounter an error, pls check your username or password');
                return;
            }
            var obj = JSON.parse(res);
            listedVideos = obj['videos'];
            obj['videos'].forEach(function(item, index) {
                var ele = htmlVideoListElement(item['id'], item['name'], item['create_time']);
                $("#items").append(ele);
            });
            callback();
        });
    }); */

    getUserId(function(res, err) {
        if (err != null) {
            window.alert("Encountered error when loading user id");
            return;
        }
        var obj = JSON.parse(res);
        uid = obj['id'];

        // 初始化默认加载 Lobby (或者你可以选 My Videos)
        $("#lobby-link").trigger('click');
        callback();
    });

}

// 封装一个通用的渲染列表函数，复用逻辑
function renderVideoList(videos) {
    listedVideos = videos;
    $("#items").empty(); // 清空当前列表
    
    if (videos && videos.length > 0) {
        $("#play-box").show();
        // 默认选中第一个
        currentVideo = videos[0];
        selectVideo(videos[0]['id']);
        
        videos.forEach(function(item, index) {
            var ele = htmlVideoListElement(item['id'], item['name'], item['create_time']);
            $("#items").append(ele);
        });
    } else {
        $("#play-box").hide();
        $("#items").append('<div style="font-size: 20px; font-weight: bold; margin-top: 20px; color: #555;">No videos found.</div>');
    }
}

function setCookie(cname, cvalue, exmin) {
    var d = new Date();
    d.setTime(d.getTime() + (exmin * 60 * 1000));
    var expires = "expires="+d.toUTCString();
    document.cookie = cname + "=" + cvalue + ";" + expires + ";path=/";
}

function getCookie(cname) {
    var name = cname + "=";
    var ca = document.cookie.split(';');
    for(var i = 0; i < ca.length; i++) {
        var c = ca[i];
        while (c.charAt(0) == ' ') {
            c = c.substring(1);
        }
        if (c.indexOf(name) == 0) {
            return c.substring(name.length, c.length);
        }
    }
    return "";
}

// DOM operations
function selectVideo(vid) {
    var url = 'http://' + window.location.hostname + ':8080/videos/'+ vid
    $("#curr-video").attr('src', url);
    $("#curr-video-name").text(currentVideo['name']);
    $("#curr-video-ctime").text('Uploaded at: ' + currentVideo['create_time']);
    //currentVideoId = vid;
    refreshComments(vid);
}

function refreshComments(vid) {
    listAllComments(vid, function (res, err) {
        if (err !== null) {
            //window.alert("encounter an error when loading comments");
            popupErrorMsg('encounter an error when loading comments');
            return
        }

        var obj = JSON.parse(res);
        $("#comments-history").empty();
        if (obj['comments'] === null) {
            $("#comments-total").text('0 Comments');
        } else {
            $("#comments-total").text(obj['comments'].length + ' Comments');
        }
        obj['comments'].forEach(function(item, index) {
            var ele = htmlCommentListElement(item['create_time'], item['authorName'], item['content']);
            $("#comments-history").append(ele);
        });

    });
}

function popupNotificationMsg(msg) {
    var x = document.getElementById("snackbar");
    $("#snackbar").text(msg);
    x.className = "show";
    setTimeout(function(){ x.className = x.className.replace("show", ""); }, 2000);
}

function popupErrorMsg(msg) {
    var x = document.getElementById("errorbar");
    $("#errorbar").text(msg);
    x.className = "show";
    setTimeout(function(){ x.className = x.className.replace("show", ""); }, 2000);
}

function htmlCommentListElement(ctime, author, content) {
    var ele = $('<div/>', {
        id: ctime
    });

    ele.append(
        $('<div/>', {
            class: 'comment-author',
            text: author + ' says:'
        })
    );
    ele.append(
        $('<div/>', {
            class: 'comment',
            text: content
        })
    );

    ele.append('<hr style="height: 1px; border:none; color:#EDE3E1;background-color:#EDE3E1">');

    return ele;
}

function htmlVideoListElement(vid, name, ctime) {
    var ele = $('<a/>', {
        href: '#'
    });
    ele.append(
        $('<video/>', {
            width:'320',
            height:'240',
            poster:'/statics/img/preloader.jpg',
            controls: true
            //href: '#'
        })
    );
    ele.append(
        $('<div/>', {
            text: name
        })
    );
    ele.append(
        $('<div/>', {
            text: ctime
        })
    );


    var res = $('<div/>', {
        id: vid,
        class: 'video-item'
    }).append(ele);

    res.append(
        $('<button/>', {
            id: 'del-' + vid,
            type: 'button',
            class: 'del-video-button',
            text: 'Delete'
        })
    );

    res.append(
        $('<hr>', {
            size: '2'
        }).css('border-color', 'grey')
    );

    return res;
}

// Async ajax methods

var apiUrl = 'http://'+window.location.hostname + ':8080/api';
// User operations
function registerUser(callback) {
    var username = $("#username").val();
    var pwd = $("#pwd").val();
    //var apiUrl = 'http://'+window.location.hostname + ':8080/api';

    if (username == '' || pwd == '') {
        callback(null, err);
    }

    var reqBody = {
        'user_name': username,
        'password': pwd
    }

    var dat = {
        'url': 'http://'+ window.location.hostname + ':8000/user',
        'method': 'POST',
        'req_body': JSON.stringify(reqBody)
    };

    $.ajax({
        url  : apiUrl,
        type : 'post',
        data : JSON.stringify(dat),
        contentType: 'application/json; charset=utf-8',
    }).done(function(data, statusText, xhr){
        if (xhr.status >= 400) {
            callback(null, "Error of register");
            return;
        }
        uname = username;
        callback(data, null);
    }).fail(function(xhr,satus,error){
        console.error("Register error:", error);
        callback(null, "Error of register");
    });
}

function signinUser(callback) {
    var username = $("#susername").val();
    var pwd = $("#spwd").val();
    //var apiUrl = window.location.hostname + ':8080/api';

    if (username == '' || pwd == '') {
        callback(null, "Username or password cannot be empty");
    }

    var reqBody = {
        'user_name': username,
        'password': pwd
    }

    var dat = {
        'url': 'http://'+ window.location.hostname + ':8000/user/' + username,
        'method': 'POST',
        'req_body': JSON.stringify(reqBody)
    };

    $.ajax({
        url  : apiUrl,
        type : 'post',
        data : JSON.stringify(dat),
        statusCode: {
            500: function() {
                callback(null, "Internal error");
            }
        },
        complete: function(xhr, textStatus) {
            if (xhr.status >= 400) {
                callback(null, "Error of Signin");
                return;
            }
        }
    }).done(function(data, statusText, xhr){
        if (xhr.status >= 400) {
            callback(null, "Error of Signin");
            return;
        }
        uname = username;
        window.console.log("Signin success");
        callback(data, null);
    });
}

function getUserId(callback) {
    var dat = {
        'url': 'http://' + window.location.hostname + ':8000/user/' + uname,
        'method': 'GET'
    };

    $.ajax({
        url: apiUrl,
        type: 'post',
        data: JSON.stringify(dat),
        headers: {
                    'X-Session-Id': getCookie("sessionid")
                },
        statusCode: {
            500: function() {
                callback(null, "Internal Error");
            }
        },
        complete: function(xhr, textStatus) {
            if (xhr.status >= 400) {
                callback(null, "Error of getUserId");
                return;
            }
        }
    }).done(function (data, statusText, xhr) {
        callback(data, null);
    });
}

// Video operations
function createVideo(vname, callback) {
    var reqBody = {
        'author_id': uid,
        'name': vname
    };

    var dat = {
        'url': 'http://' + window.location.hostname + ':8000/user/' + uname + '/videos',
        'method': 'POST',
        'req_body': JSON.stringify(reqBody)
    };

    $.ajax({
        url  : apiUrl,
        type : 'post',
        data : JSON.stringify(dat),
        headers: {
                    'X-Session-Id': getCookie("sessionid")
                },
        statusCode: {
            500: function() {
                callback(null, "Internal error");
            }
        },
        complete: function(xhr, textStatus) {
            if (xhr.status >= 400) {
                callback(null, "Error of Signin");
                return;
            }
        }
    }).done(function(data, statusText, xhr){
        if (xhr.status >= 400) {
            callback(null, "Error of Signin");
            return;
        }
        callback(data, null);
    });
}

function listLobbyVideos(callback) {
    var dat = {
        'url': 'http://' + window.location.hostname + ':8000/videos',
        'method': 'GET',
        'req_body': ''
    };

    $.ajax({
        url  : apiUrl,
        type : 'post',
        data : JSON.stringify(dat),
        headers: {
                    'X-Session-Id': getCookie("sessionid")
                },
        statusCode: {
            500: function() {
                callback(null, "Internal error");
            }
        },
        complete: function(xhr, textStatus) {
            if (xhr.status >= 400) {
                callback(null, "Error of listLobbyVideos");
                return;
            }
        }
    }).done(function(data, statusText, xhr){
        if (xhr.status >= 400) {
            callback(null, "Error of listLobbyVideos");
            return;
        }
        callback(data, null);
    });
}

function listAllUserVideos(callback) {
    var dat = {
        'url': 'http://' + window.location.hostname + ':8000/user/' + uname + '/videos',
        'method': 'GET',
        'req_body': ''
    };

    $.ajax({
        url  : apiUrl,
        type : 'post',
        data : JSON.stringify(dat),
        headers: {
                    'X-Session-Id': getCookie("sessionid")
                },
        statusCode: {
            500: function() {
                callback(null, "Internal error");
            }
        },
        complete: function(xhr, textStatus) {
            if (xhr.status >= 400) {
                callback(null, "Error of Signin");
                return;
            }
        }
    }).done(function(data, statusText, xhr){
        if (xhr.status >= 400) {
            callback(null, "Error of Signin");
            return;
        }
        callback(data, null);
    });
}

function deleteVideo(vid, callback) {
    var dat = {
        'url': 'http://' + window.location.hostname + ':8000/user/' + uname + '/videos/' + vid,
        'method': 'DELETE',
        'req_body': ''
    };

    $.ajax({
        url  : apiUrl,
        type : 'post',
        data : JSON.stringify(dat),
        headers: {
                    'X-Session-Id': getCookie("sessionid")
                },
        statusCode: {
            500: function() {
                callback(null, "Internal error");
            }
        },
        complete: function(xhr, textStatus) {
            if (xhr.status >= 400) {
                callback(null, "Error of Signin");
                return;
            }
        }
    }).done(function(data, statusText, xhr){
        if (xhr.status >= 400) {
            callback(null, "Error of Signin");
            return;
        }
        callback(data, null);
    });
}

// Comments operations
function postComment(vid, content, callback) {
    var reqBody = {
        'author_id': uid,
        'content': content
    }
    var dat = {
        'url': 'http://' + window.location.hostname + ':8000/videos/' + vid + '/comments',
        'method': 'POST',
        'req_body': JSON.stringify(reqBody)
    };

    $.ajax({
        url  : apiUrl,
        type : 'post',
        data : JSON.stringify(dat),
        headers: {
                    'X-Session-Id': getCookie("sessionid")
                },
        statusCode: {
            500: function() {
                callback(null, "Internal error");
            }
        },
        complete: function(xhr, textStatus) {
            if (xhr.status >= 400) {
                callback(null, "Error of Signin");
                return;
            }
        }
    }).done(function(data, statusText, xhr){
        if (xhr.status >= 400) {
            callback(null, "Error of Signin");
            return;
        }
        callback(data, null);
    });
}

function listAllComments(vid, callback) {
    var dat = {
        'url': 'http://' + window.location.hostname + ':8000/videos/' + vid + '/comments',
        'method': 'GET',
        'req_body': ''
    };

    $.ajax({
        url  : apiUrl,
        type : 'post',
        data : JSON.stringify(dat),
        headers: {
                    'X-Session-Id': getCookie("sessionid")
                },
        statusCode: {
            500: function() {
                callback(null, "Internal error");
            }
        },
        complete: function(xhr, textStatus) {
            if (xhr.status >= 400) {
                callback(null, "Error of Signin");
                return;
            }
        }
    }).done(function(data, statusText, xhr){
        if (xhr.status >= 400) {
            callback(null, "Error of Signin");
            return;
        }
        callback(data, null);
    });
}