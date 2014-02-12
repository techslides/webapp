//your main js file

$(function(){

  //delete a user
  $(".delete_user").click(function(){
    var id = $(this).attr("rel");
    var url = '/users/'+id;
    $.ajax({
      url: url,
      type: 'DELETE',
      success: function(result) {
          //we can do some error checking here, as in ajax was good but DB was not a success
          if(result.responseText != "success"){
            $("#errors").text("Something went wrong. User was not deleted.")
          } else { 
            window.location.replace("/");
          }
      },
      error: function(result) {
          //general div to handle error messages
          $("#errors").text(result.responseText);
      }
    });
  });

  //delete a post
  $(".delete_post").click(function(){
    var id = $(this).attr("rel");
    var url = '/posts/'+id;
    $.ajax({
      url: url,
      type: 'DELETE',
      success: function(result) {
          //we can do some error checking here, as in ajax was good but DB was not a success
          if(result.responseText != "success"){
            $("#errors").text("Something went wrong. Post was not deleted.")
          } else { 
            window.location.replace("/");
          }
      },
      error: function(result) {
          //general div to handle error messages
          $("#errors").text(result.responseText);
      }
    });
  });




  //modifying user data
  $(".useredit").on("blur", function() {
    var id = $("#userid").val();
    var url = '/users/'+id;

    var obj = {};
    obj.name = $('#username').text();
    obj.email = $('#useremail').text();
    obj.password = $('#userpassword').text();

    ajax_put(url,obj); 
  });

  //modifying post data
  $(".postedit").on("blur", function() {
    var id = $("#postid").val();
    var url = '/posts/'+id;

    var obj = {};
    obj.Title = $('#posttitle').text();
    obj.Body = $('#postbody').html();

    ajax_put(url,obj);
  });


  //PUT Ajax calls
  function ajax_put(url,obj){
    $.ajax({
      url: url,
      type: 'PUT',
      data: obj,
      success: function(result) {
          console.log(result);
	  //we can do some error checking here, as in ajax was good but DB was not a success
	  if(result.responseText != "success"){
            $("#errors").text("Something went wrong. User was not updated.")
	  }
      },
      error: function(result) {
          //general div to handle error messages
          $("#errors").text(result.responseText);
          //if martini binding recognizes a validation error, this is where you can decipher the JSON and properly display that shit
          // {"overall":{},"fields":{"Title":"Required","title":"Title cannot be empty"}}
          // JSON.parse(result.responseText).fields.title
      }
    });
  }


});
