
var fragments = {};

fragments.db = {};
if (window.localStorage) {
   fragments.db = window.localStorage;
}

fragments.log = function(message) {
   if (message) {
      console.log("fragments: " + message);
   } else {
      console.log("");
   }
}

fragments.getHtml = function(id) {
   if (id in fragments.db) {
      var item = JSON.parse(fragments.db[id]);
      return item.Html;
   }
   return null;
}

fragments.store = function(url, item) {
   var id = item.Id;
   if (id == '_root') {
      id += url;
   }
   fragments.db[id] = JSON.stringify({
      Html: item.Html,
      Stamp: item.Stamp,
   });
}

fragments.replace = function(item) {
   if (item.Html === "") {
      item.Html = fragments.getHtml(item.Id);
   } else {
      fragments.log('replace("' + item.Id + '")');
   }
   var elem = $('div[fragment="' + item.Id + '"]');
   var children = {}
   $(elem).find('div[fragment]').each(function () {
      var id = $(this).attr("fragment");
      children[id] = $(this).detach();
   });
   $(elem).html(item.Html);
   for (var id in children) {
      if (children.hasOwnProperty(id)) {
         $(elem)
            .find('div[fragment="' + id + '"]')
            .replaceWith(children[id]);
      }
   }
}

fragments.getStamp = function(url) {
   fragments.log('getStamp("' + url + '")');
   var id = '_root' + url;
   if (fragments.db[id]) {
      var item = JSON.parse(fragments.db[id]);
      fragments.log("  " + item.Stamp);
      return item.Stamp
   }
   fragments.log("  null");
   return null;
}

fragments.assembleFull = function(url, list) {
   fragments.log("assembleFull(");
   for (var i = 0; i < list.length; i++) {
      fragments.log("  [" + list[i].Stamp + "] " + list[i].Id);
      $('div[fragment="' + list[i].Id + '"]').html(list[i].Html);
      fragments.store(url, list[i]);
   }
   fragments.log(")");
}

fragments.assemblePartial = function(url, list) {
   var msg = "assemblePartial({";
   for (var i = 0; i < list.length; i++) {
      msg += " " + list[i].Id;
      fragments.replace(list[i])
      fragments.store(url, list[i]);
   }
   msg += " })";
   fragments.log(msg);
}

fragments.loadFull = function(url) {
   fragments.log('loadFull("' + url + '")');
   $.ajax({
      url: url,
      headers: {"Fragments": "all"}, 
      dataType:"json"
   }).done(function(list) {
      fragments.assembleFull(url, list);
      fragments.replaceLinks();
   });
}

fragments.loadPartial = function(url, stamp) {
   fragments.log('loadPartial("' + url + '", ' + stamp + ')');
   $.ajax({
      url: url,
      headers: {
         "Fragments": "since",
         "FragmentsStamp": '"' + stamp + '"',
      }, 
      dataType:"json"
   }).done(function(list) {
      fragments.assemblePartial(url, list);
      fragments.replaceLinks();
   });
}

fragments.load = function(url) {
   var stamp = fragments.getStamp(url);
   if (stamp) {
      fragments.loadPartial(url, stamp);
   } else {
      fragments.loadFull(url);
   }
   $('html, body').scrollTop(0); // Go to top
}

fragments.follow = function(link) {
   var url = $(link).attr("href")
   fragments.log("follow: " + url);
   fragments.load(url);
   history.pushState(url, null, url);
}

fragments.replaceLinks = function() {
   fragments.log("replaceLinks()");
   $('a[ajx]').click(function (ev) {
      ev.preventDefault();
      fragments.follow(this);
   });
   fragments._notify();
   fragments.log();
}

fragments._fns = []

fragments.ready = function(fn) {
   fragments._fns.push(fn);
}

fragments._notify = function() {
   var fns = fragments._fns;
   for (var i = 0; i < fns.length; i++) {
      fns[i]();
   }
}

// Bugs:
// - Chrome: Al clicar a la '/' manualmente (el icono de 'academ.io')
//           el botón 'back' muestra el texto AJAX... (en Chrome)

fragments._onpopstate = function(ev) {
   fragments.log("_onpopstate(" + JSON.stringify(ev.state) + ")")
   fragments.log();
   if (ev.state === null) {
      history.replaceState("reload", null, document.location.pathname);
   } else if (ev.state === "reload") {
      document.location.reload();
   } else {
      fragments.load(ev.state);
   }
}

$(document).ready(function () {
   history.replaceState("reload", null, document.location.pathname);
   fragments.log("ready()");
   if (history && history.pushState) {
      fragments.replaceLinks();
      onpopstate = fragments._onpopstate;
   }
})
