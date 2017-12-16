var app = angular.module('app', ['angularMoment']);

app.controller('cleo', ['$scope', function($scope) {
		$scope.Delalerts = function(){
			$scope.alerts = []
			DeleteAlerts(function(){

			})
		}
		$scope.alerts = [];
		$scope.Refresh = function(){
			$scope.working = true
			Cleo( function(data,success){
				$scope.working = false;
				$scope.alerts = data.cleo.Alerts.reverse();
				$scope.$apply();
			} )
		}

		$scope.Refresh

		setInterval(function(){
			$scope.Refresh()
		}, 2000)
}])


//momentum dep handler
function jsrequestmomentum(url,payload,type,callback){
   var xhttp = new XMLHttpRequest();
  	xhttp.onreadystatechange = function() {
  		if(xhttp.readyState == 4){
   		var success = ( xhttp.status == 200)
    	if (type == "POSTJSON"){
    		try {
    		callback(JSON.parse(xhttp.responseText), success);
    		} catch (e) {
    			console.log("Invalid JSON");
    			callback({error : xhttp.responseText == "" ? "Server wrote no response" : xhttp.responseText}, false )
    		}
    	} else callback(xhttp.responseText, success );
    }
  };

  var serialize = function(obj) {
  var str = [];
  for(var p in obj)
    if (obj.hasOwnProperty(p)) {
      str.push(encodeURIComponent(p) + "=" + encodeURIComponent(obj[p]));
    }
  return str.join("&");
  }
  xhttp.open(type, url, true);

  if(type == "POST"){
    xhttp.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
    xhttp.send(serialize(payload));
} else   if(type == "POSTJSON"){
    xhttp.setRequestHeader("Content-type", "application/json");
    xhttp.send(JSON.stringify(payload));
}  else  xhttp.send();
}