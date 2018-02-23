var MinutesB = function(date1, date2) {
    //Get 1 day in milliseconds
    var one_day = 1000 * 60;

    // Convert both dates to milliseconds
    var date1_ms = date1.getTime();
    var date2_ms = date2.getTime();

    // Calculate the difference in milliseconds
    var difference_ms = date2_ms - date1_ms;

    // Convert back to days and return
    return Math.round(difference_ms / one_day);
}

var lineuChart, lineoChart, DoughnutChart;

var formatBytes = function(bytes, decimals) {
    if (bytes == 0) return '0 Bytes';
    var k = 1024,
        dm = decimals || 2,
        i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm));
}

var app = angular.module('app', ['angularMoment']);

app.controller('cleo-app', ['$scope', function($scope) {
    $scope.app = {}
    $scope.alert = false
    $scope.working = false
    $scope.Add = function() {
        $scope.working = true
        AddApp($scope.app, function(data, success) {
            $scope.working = false
            $scope.alert = true
            $scope.app = { Name: "" }
            $scope.$apply()
        })
    }
}])

app.controller('cleo-test', ['$scope', function($scope) {
    $scope.test = {}
    $scope.alert = false
    $scope.working = false
    $scope.apps = [];

    $scope.Refresh = function() {
        $scope.working = true
        Cleo(function(data, success) {
            $scope.working = false
            $scope.alert.text = ""
            if (data.cleo.Apps)
                $scope.apps = data.cleo.Apps;

            $scope.$apply()
        })
    }

    $scope.Refresh()

    $scope.Add = function() {
        $scope.working = true
        AddTest($scope.test, function(data, success) {
            $scope.working = false
            $scope.alert = true
            $scope.test = { Name: "" }
            $scope.$apply()
        })
    }

}])


app.controller('cleo-settings', ['$scope', function($scope) {
    $scope.settings = {}
    $scope.working = false
    $scope.alert = { danger: false };
    $scope.confirmdel = false;

    $scope.Refresh = function() {
        $scope.working = true
        Cleo(function(data, success) {
            $scope.working = false

            $scope.procMessage(data, success)
            $scope.alert.text = ""
            $scope.settings = data.cleo.Settings
            $scope.$apply()
        })
    }

    $scope.Nuke = function() {
        if (!$scope.confirmdel) {
            $scope.confirmdel = true;
            setTimeout(function() {
                $scope.confirmdel = false;
                $scope.$apply()
            }, 6000)
            return
        }
        Nuke(function(data, success) {
            $scope.procMessage(data, success)
        })
    }

    $scope.Update = function() {
        $scope.working = true;
        UpdateSettings($scope.settings, function(data, success) {
            $scope.working = false;
            $scope.procMessage(data, success)
            $scope.$apply()
        })
    }


    $scope.procMessage = function(data, success) {
        if (!success) {
            $scope.alert.text = data.error
            $scope.alert.danger = true
            return
        } else {
            $scope.alert.text = "Changes performed."
            $scope.alert.danger = false
            return
        }
    }

    $scope.Refresh()

}])

app.controller('cleo-apps', ['$scope', function($scope) {
    $scope.apps = []
    $scope.nenv = {}
    $scope.working = false
    $scope.alert = { danger: false };
    $scope.confirmdel = false;

    $scope.Refresh = function() {
        $scope.working = true
        Cleo(function(data, success) {
            $scope.working = false
            $scope.procMessage(data, success)
            $scope.alert.text = ""
            $scope.apps = data.cleo.Apps
            $scope.edit = {}
            $scope.$apply()
        })
    }

    $scope.Edit = function(app) {
        $scope.edit = app;
    }

    $scope.DeleteEnv = function(target) {
        $scope.edit.Envs.splice($scope.edit.Envs.indexOf(target), 1)
    }

    $scope.AddEnv = function() {
        if (!$scope.edit.Envs) $scope.edit.Envs = []
        $scope.edit.Envs.push($scope.nenv)
        $scope.nenv = {}
    }

    $scope.procMessage = function(data, success) {
        if (!success) {
            $scope.alert.text = data.error
            $scope.alert.danger = true
            return
        } else {
            $scope.alert.text = "Changes performed."
            $scope.alert.danger = false
            return
        }
    }

    $scope.UpdateApp = function() {
        $scope.working = true
        UpdateApp($scope.edit, function(data, success) {
            $scope.working = false
            $scope.procMessage(data, success)
            $scope.$apply()
        })
    }
    $scope.DeleteApp = function() {
        if (!$scope.confirmdel) {
            $scope.confirmdel = true;
            setTimeout(function() {
                $scope.confirmdel = false;
                $scope.$apply()
            }, 6000)
            return
        }
        $scope.working = true
        DeleteApp($scope.edit, function(data, success) {
            $scope.working = false
            $scope.procMessage(data, success)
            $scope.confirmdel = false;
            $scope.apps.splice($scope.apps.indexOf($scope.edit), 1)
            $scope.edit = {};
            $scope.$apply();
        })
    }
    $scope.Refresh()
}])

app.controller('cleo-tests', ['$scope', function($scope) {
    $scope.tests = []
    $scope.nenv = {}
    $scope.apps = []
    $scope.working = false
    $scope.alert = { danger: false };
    $scope.confirmdel = false;

    $scope.Refresh = function() {
        $scope.working = true
        Cleo(function(data, success) {
            $scope.working = false
            $scope.procMessage(data, success)
            $scope.alert.text = ""
            if (data.cleo.Tests)
                $scope.tests = data.cleo.Tests
            if (data.cleo.Apps)
                $scope.apps = data.cleo.Apps
            $scope.edit = {}
            $scope.$apply()
        })
    }
    $scope.infilter = function(name){
    	return name.includes($scope.report.filter);
    }
    $scope.Edit = function(app) {
        $scope.edit = app;
        $scope.report = {filter :""}
        if (lineuChart) {
            lineuChart.destroy();
            lineoChart.destroy();
            DoughnutChart.destroy();

        }
        if ($scope.edit.HeapMinute) {
            var dates = []
            var huse = {
                label: 'Heap in use.',
                data: [],
                borderWidth: 1,
                fill: true,
                lineTension: 0.1,
                backgroundColor: "red",
                borderColor: "red",
            }
            var hrl = {
                label: 'Heap released.',
                data: [],
                borderWidth: 1,
                fill: true,
                lineTension: 0.1,
                backgroundColor: "green",
                borderColor: "green",
            }
            var maxCpu = {
                label: 'Maximum CPU time',
                data: [],
                borderWidth: 1,
                fill: false,
                lineTension: 0.1,
                backgroundColor: "red",
                borderColor: "red",
            }
            var actCPU = {
                label: 'Actual CPU time',
                data: [],
                borderWidth: 1,
                fill: true,
                lineTension: 0.1,
                backgroundColor: "orange",
                borderColor: "orange",
            }
            var ho = {
                label: 'Objects in heap.',
                data: [],
                borderWidth: 1,
                fill: true,
                lineTension: 0.1,
                backgroundColor: "orange",
                borderColor: "orange",
            }
            for (var i = $scope.edit.HeapMinute.length - 1; i >= 0; i--) {
                var heap = $scope.edit.HeapMinute[i];
                //Iu Rl Ho
                dates.push(heap.Time)
                huse.data.push(formatBytes(heap.Iu, 4))
                hrl.data.push(formatBytes(heap.Rl, 4))
                ho.data.push(heap.Ho)
            }

            var dset = [huse, hrl]


            GetCard(app, function(data) {
                $scope.report.card = data.res;
                $scope.$apply();
            })

            if ($scope.edit.CPU)
                GetCPUtimes(app, function(data) {
                    for (var i = data.top.length - 1; i >= 0; i--) {
                        var frame = data.top[i];
                        maxCpu.data.push($scope.edit.MaxCPU);
                        actCPU.data.push(frame.CPUUsage);
                    }




                    var cpuset = [maxCpu, actCPU];

                    var el = document.getElementById("reportCPUChart");
                    el.style.height = "150px";
                    var ctx = el.getContext('2d');
                    lineuChart = new Chart(ctx, {
                        type: 'line',
                        data: {
                            labels: dates,
                            datasets: cpuset
                        },
                        options: {
                            scales: {
                                xAxes: [{
                                    type: 'time'
                                }],
                                yAxes: [{
                                    gridLines: {
                                        color: "gray",
                                        borderDash: [2, 5],
                                    },
                                    scaleLabel: {
                                        display: true,
                                        labelString: "Time in MS",
                                        fontColor: "black"
                                    }
                                }]
                            }
                        }
                    });

                });

            if ($scope.edit.CPU)
                GetCPUTop(app, function(data) {
                    $scope.report.topCPU = data.top
                    $scope.report.totaltime = '' + data.total;
                    var ctx = document.getElementById("topCPUChart").getContext('2d');

                    var lbls = []
                    var dagg = []

                    for (var i = data.top.length - 1; i >= 0; i--) {
                        var top = data.top[i];
                        lbls.push(top.Name)
                        dagg.push(top.Percent)
                    }
                    var data = {
                        datasets: [{
                            data: dagg,
                            backgroundColor: ["#333", "#0074D9", "#FF4136", "#2ECC40", "#FF851B", "#7FDBFF", "#B10DC9", "#FFDC00", "#001f3f", "#39CCCC", "#01FF70", "#85144b", "#F012BE", "#3D9970", "#111111", "#AAAAAA"]
                        }],

                        // These labels appear in the legend and in the tooltips when hovering different arcs
                        labels: lbls
                    };
                    DoughnutChart = new Chart(ctx, {
                        type: 'doughnut',
                        data: data,
                        options: {}
                    });

                    $scope.$apply()
                })

            GetTop(app, function(data) {
                $scope.report.top = data.top

                var ctx = document.getElementById("topChart").getContext('2d');

                var lbls = []
                var dagg = []

                for (var i = data.top.length - 1; i >= 0; i--) {
                    var top = data.top[i];
                    lbls.push(top.Name)
                    dagg.push(top.Percent)
                }
                var data = {
                    datasets: [{
                        data: dagg,
                        backgroundColor: ["#333", "#0074D9", "#FF4136", "#2ECC40", "#FF851B", "#7FDBFF", "#B10DC9", "#FFDC00", "#001f3f", "#39CCCC", "#01FF70", "#85144b", "#F012BE", "#3D9970", "#111111", "#AAAAAA"]
                    }],

                    // These labels appear in the legend and in the tooltips when hovering different arcs
                    labels: lbls
                };
                DoughnutChart = new Chart(ctx, {
                    type: 'doughnut',
                    data: data,
                    options: {}
                });

                $scope.$apply()
            })
            $scope.report.duration = MinutesB(new Date(app.Start), new Date(app.End))

            setTimeout(function() {
                var ctx = document.getElementById("reportChart").getContext('2d');
                lineuChart = new Chart(ctx, {
                    type: 'line',
                    data: {
                        labels: dates,
                        datasets: dset
                    },
                    options: {
                        scales: {
                            xAxes: [{
                                type: 'time'
                            }],
                            yAxes: [{
                                gridLines: {
                                    color: "gray",
                                    borderDash: [2, 5],
                                },
                                scaleLabel: {
                                    display: true,
                                    labelString: "Size in Kb",
                                    fontColor: "black"
                                }
                            }]
                        }
                    }
                });

                ctx = document.getElementById("objChart").getContext('2d');
                lineoChart = new Chart(ctx, {
                    type: 'line',
                    data: {
                        labels: dates,
                        datasets: [ho]
                    },
                    options: {
                        scales: {
                            xAxes: [{
                                type: 'time'
                            }],
                            yAxes: [{
                                gridLines: {
                                    color: "gray",
                                    borderDash: [2, 5],
                                },
                                scaleLabel: {
                                    display: true,
                                    labelString: "Object count",
                                    fontColor: "black"
                                }
                            }]
                        }
                    }
                });
            }, 1000)
        }
    }

    $scope.List = function(item) {
        $scope.report.current = item
        $scope.working = true
        $scope.report.list = "Loading"
        GetList($scope.edit, item, function(data) {
            $scope.working = false
            $scope.report.list = data.list
            if (data.list == "") {
                $scope.report.list = "Nothing found."
            }
            $scope.$apply()
        })
    }

    $scope.CPUList = function(item) {
        $scope.report.currentSample = item
        $scope.working = true
        $scope.report.listCPU = "Loading"
        GetListCPU($scope.edit, item, function(data) {
            $scope.working = false
            $scope.report.listCPU = data.list
            if (data.list == "") {
                $scope.report.list = "Nothing found."
            }
            $scope.$apply()
        })
    }

    $scope.procMessage = function(data, success) {
        if (!success) {
            $scope.alert.text = data.error
            $scope.alert.danger = true
            return
        } else {
            $scope.alert.text = "Changes performed."
            $scope.alert.danger = false
            return
        }
    }

    $scope.Start = function() {
        $scope.working = true
        Start($scope.edit, function(data, success) {
            $scope.working = false
            $scope.procMessage(data, success)
            $scope.edit.Start = new Date()
            $scope.edit.Working = success

            $scope.$apply()
        })
    }

    $scope.Cancel = function() {
        $scope.working = true
        $scope.edit.Finished = true;
        $scope.edit.Working = false
        Cancel($scope.edit, function(data, success) {
            $scope.working = false
            $scope.procMessage(data, success)
            $scope.edit.End = new Date()


            $scope.$apply()
        })
    }
    $scope.UpdateTest = function() {
        $scope.working = true
        UpdateTest($scope.edit, function(data, success) {
            $scope.working = false
            $scope.procMessage(data, success)
            $scope.$apply()
        })
    }
    $scope.DeleteTest = function() {
        if (!$scope.confirmdel) {
            $scope.confirmdel = true;
            setTimeout(function() {
                $scope.confirmdel = false;
                $scope.$apply()
            }, 6000)
            return
        }
        $scope.working = true
        DeleteTest($scope.edit, function(data, success) {
            $scope.working = false
            $scope.procMessage(data, success)
            $scope.confirmdel = false;
            $scope.tests.splice($scope.tests.indexOf($scope.edit), 1)
            $scope.edit = {};
            $scope.$apply();
        })
    }
    $scope.Refresh()
}])

//momentum dep handler
function jsrequestmomentum(url, payload, type, callback) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (xhttp.readyState == 4) {
            var success = (xhttp.status == 200)
            if (type == "POSTJSON") {
                try {
                    callback(JSON.parse(xhttp.responseText), success);
                } catch (e) {
                    console.log("Invalid JSON");
                    callback({ error: xhttp.responseText == "" ? "Server wrote no response" : xhttp.responseText }, false)
                }
            } else callback(xhttp.responseText, success);
        }
    };

    var serialize = function(obj) {
        var str = [];
        for (var p in obj)
            if (obj.hasOwnProperty(p)) {
                str.push(encodeURIComponent(p) + "=" + encodeURIComponent(obj[p]));
            }
        return str.join("&");
    }
    xhttp.open(type, url, true);

    if (type == "POST") {
        xhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
        xhttp.send(serialize(payload));
    } else if (type == "POSTJSON") {
        xhttp.setRequestHeader("Content-type", "application/json");
        xhttp.send(JSON.stringify(payload));
    } else xhttp.send();
}