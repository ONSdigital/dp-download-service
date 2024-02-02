// find download links (href,public) in mongo instances
// that contain old domains/buckets,
// replace the link with the correct domain/bucket
//
// does at-most `lim` of each type of conversion, per run, so,
// you probably want to amend that value, or run several times
// until count_done=0 is the output
//
// mongo --quiet mongodb://user:pass@127.0.0.1:27017/datasets?authSource=admin fix-download-links.js
//
// note the `/datasets` in the above

instances_coll = db.getCollection('instances');
download_formats = { 'xlsx': 1, 'xls': 1, 'csv': 1, 'csvw': 1 };      // keys are keys of the `downloads` sub-doc
debugging = true;       // when `false`, will change the db!
lim = 10;
count_done = 0;

function replace_link(key, oldv, newv) {
  var o = [{}, { '$set': {} }];
  o[0][key] = oldv;
  o[1]['$set'][key] = newv;
  printjson(o);
  if (!debugging) {
    instances_coll.findOneAndUpdate(o[0], o[1])
  }
  count_done++;
}

for (dl_fmt in download_formats) {
  print('processing: ' + dl_fmt);

  q = {};
  view = {};
  k = 'downloads.' + dl_fmt + '.href';
  q[k] = /onsdigital/;
  view[k] = true;
  cursor = instances_coll.find(q, view).limit(lim);
  while (cursor.hasNext()) {
    doc = cursor.next();
    val = doc.downloads[dl_fmt].href;
    newval = val.replace('//download.cmd.onsdigital.co.uk/', '//download.ons.gov.uk/');
    replace_link(k, val, newval);
  }

  // -------

  q = {};
  view = {};
  k = 'downloads.' + dl_fmt + '.href';
  q[k] = /download.beta.ons/;
  view[k] = true;
  cursor = instances_coll.find(q, view).limit(lim);
  while (cursor.hasNext()) {
    doc = cursor.next();
    val = doc.downloads[dl_fmt].href;
    newval = val.replace('//download.beta.ons.', '//download.ons.');
    replace_link(k, val, newval);
  }

  // -------

  q = {};
  view = {};
  k = 'downloads.' + dl_fmt + '.public';
  q[k] = /static-cmd\.s3/;
  view[k] = true;
  cursor = instances_coll.find(q, view).limit(lim);
  while (cursor.hasNext()) {
    doc = cursor.next();
    val = doc.downloads[dl_fmt].public;
    newval = val.replace('//static-cmd.s3', '//ons-dp-production-static.s3');
    replace_link(k, val, newval);
  }

}

print('count_done: ' + count_done);
