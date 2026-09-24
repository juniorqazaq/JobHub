package ingestion

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"jobhub-ai/backend/internal/config"
	"jobhub-ai/backend/internal/database"
	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers/website"
)

func TestWebsiteImportUpdatesAndPreservesUUID(t *testing.T){
	raw:=os.Getenv("TEST_DATABASE_URL");if raw==""{t.Skip("TEST_DATABASE_URL is not set")};if config.ValidateWebsiteDevelopmentDatabase("development",raw)!=nil{t.Skip("website integration requires isolated local jobhub_website_poc_* database")}
	ctx:=context.Background();pool,err:=database.Connect(ctx,raw,"cache_statement");if err!=nil{t.Fatal(err)};t.Cleanup(pool.Close);store:=jobs.NewPostgresStore(pool);source:="website:integration.example";if err:=store.RegisterDevelopmentSource(ctx,source,"website","Integration Example");err!=nil{t.Fatal(err)};t.Cleanup(func(){_,_=pool.Exec(ctx,"delete from jobhub.jobs where source=$1",source);_,_=pool.Exec(ctx,"delete from jobhub.ingestion_runs where source=$1",source);_,_=pool.Exec(ctx,"delete from jobhub.job_sources where source=$1",source)})
	base:=jobs.ImportedJob{Source:source,ExternalID:"https://integration.example/jobs/1",SourceURL:"https://integration.example/jobs/1",Title:"Engineer",Description:"Build",DescriptionKind:"full"};svc:=NewService(store,nil,slog.New(slog.NewTextHandler(io.Discard,nil)),0);now:=time.Now().UTC().Truncate(time.Microsecond);svc.now=func()time.Time{return now};first,err:=svc.Run(ctx,website.Collection{SourceName:source,Items:[]jobs.ImportedJob{base},Requests:2},RunOptions{NoAutomaticExpiry:true});if err!=nil||first.Stats.InsertedCount!=1{t.Fatalf("first: %+v %v",first,err)}
	var id string;var firstSeen time.Time;if err:=pool.QueryRow(ctx,"select id::text,first_seen_at from jobhub.jobs where source=$1",source).Scan(&id,&firstSeen);err!=nil{t.Fatal(err)};base.Title="Senior Engineer";svc.now=func()time.Time{return now.Add(time.Minute)};second,err:=svc.Run(ctx,website.Collection{SourceName:source,Items:[]jobs.ImportedJob{base},Requests:2},RunOptions{NoAutomaticExpiry:true});if err!=nil||second.Stats.InsertedCount!=0||second.Stats.UpdatedCount!=1{t.Fatalf("second: %+v %v",second,err)}
	var gotID,title,method,apply string;var gotFirst time.Time;var count int;var unknowns bool;if err:=pool.QueryRow(ctx,`select min(id::text),min(title),min(application_method),min(apply_url),min(first_seen_at),count(*),bool_and(company_name_raw is null and location_raw is null and salary_raw is null and work_mode is null and experience_level is null) from jobhub.jobs where source=$1`,source).Scan(&gotID,&title,&method,&apply,&gotFirst,&count,&unknowns);err!=nil{t.Fatal(err)};if gotID!=id||title!="Senior Engineer"||method!="external"||apply!=base.SourceURL||!gotFirst.Equal(firstSeen)||count!=1||!unknowns{t.Fatalf("identity/external/null proof failed: id=%s title=%s method=%s apply=%s count=%d null=%v",gotID,title,method,apply,count,unknowns)}
}
