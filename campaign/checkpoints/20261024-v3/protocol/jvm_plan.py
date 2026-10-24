"""Print a portable author-JVM plan. Does not create directories or run commands."""
from pathlib import Path
import argparse,json
B=Path(__file__).resolve().parents[1]
def main():
 ap=argparse.ArgumentParser(description=__doc__);ap.add_argument('--app',choices=['gson','supplier'],required=True);ap.add_argument('--prepared-root',type=Path,required=True);ap.add_argument('--work',type=Path,required=True);ap.add_argument('--jdk-home',type=Path,required=True);a=ap.parse_args()
 prepared=a.prepared_root.resolve();work=a.work.resolve();jdk=a.jdk_home.resolve();assert not work.exists(),'New external work path required';assert work!=B and B not in work.parents
 group=json.loads((prepared/'PREPARED-SOURCES.json').read_text())['groups'][a.app];contract=json.loads((B/f'contracts/{a.app}-author-jvm.json').read_text());classes=work/'classes';empty=work/'empty';steps=[]
 bind=lambda value:value.replace('@CLASSES@',str(classes)).replace('@EMPTY@',str(empty)).replace('@WORK@',str(work))
 steps.append({'label':'javac-full-source-closure','kind':'build','cap':300,'cwd':str(work/'compile-cwd'),'argv':[str(jdk/'bin/javac'),*[bind(value)for value in contract['javac_flags_template']],*[str(prepared/row['path'])for row in group['sources']]],'setup':['Create new work/empty/classes/compile-cwd/tmp directories.','Hash every recovered source and JDK executable/release before creating a child.'],'post_compile_resource_copies':group['resources']if a.app=='supplier'else{}})
 for seed in[17,41,97]:
  for repeat in[1,2,3]:
   label=f'seed{seed}-repeat{repeat}';steps.append({'label':label,'kind':'exec','cap':60,'cwd':str(work/'run-cwd'/label),'argv':[str(jdk/'bin/java'),*[bind(value)for value in contract['JVM_flags_template']],contract['main'],*contract['seed_argv'][str(seed)]],'fresh_cwd_resource_copies':group['resources']if a.app=='gson'else{},'declared_created_files':contract['declared_created_files'],'measured_oracle_record':f'author-oracles/{a.app}/INDEX.json#{seed}'})
 print(json.dumps({'status':'PLAN_ONLY_REQUIRES_NEW_ROOT_LANE_AND_REPINNING_NO_EXECUTION_AUTHORITY','app':a.app,'source_count':group['source_count'],'steps':steps,'disk_floor_bytes':10737418240,'CPU':2,'JDK_heap':'1g','no_Go_or_acceptance_activity':True,'lifecycle':'Use bounded_run.py with unchanged LC866 supervision, durable STARTED-before-Popen census, actual launcher/controller wait/reap and independent final PID/group absence. All original pins and measured observations must be rechecked.','source_change_invalidates_all_nine_oracles':True},sort_keys=True,indent=2))
if __name__=='__main__':main()
